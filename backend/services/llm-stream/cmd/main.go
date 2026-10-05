package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	amqp "github.com/rabbitmq/amqp091-go"

	"inkwords-backend/services/llm-stream/app/bootstrap"
	textbookapp "inkwords-backend/services/llm-stream/app/textbook"
	streamdomain "inkwords-backend/services/llm-stream/domain/stream"
	"inkwords-backend/services/llm-stream/infra/rejecteddraft"
	sharedgeneration "inkwords-backend/shared/kernel/generation"
	"inkwords-backend/shared/kernel/httpx"
	platformllm "inkwords-backend/shared/platform/llm"
	sharedrabbitmq "inkwords-backend/shared/platform/rabbitmq"
)

const (
	defaultTextbookGenerationRequestTimeout = 15 * time.Minute
	minTextbookGenerationRequestTimeout     = 30 * time.Second
	maxTextbookGenerationRequestTimeout     = 30 * time.Minute
)

func init() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using default environment variables")
	}
}

func main() {
	router, taskService, streamService, workspaceID, err := bootstrap.BuildRouter()
	if err != nil {
		log.Fatalf("bootstrap llm-stream failed: %v", err)
	}
	router.POST("/api/v1/stream/provider-connection-test", providerConnectionHandler(providerConnectionTesterFromEnvironment))

	server := httpx.NewServer(router)
	signalContext, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	stopConsumer, err := startGenerationTaskConsumer(signalContext, taskService, streamService, workspaceID)
	if err != nil {
		log.Printf("RabbitMQ generation consumer initialization skipped: %v", err)
	}
	defer stopConsumer()

	go func() {
		if err := httpx.ShutdownOnContextDone(signalContext, server, 15*time.Second); err != nil {
			log.Printf("Server shutdown failed: %v", err)
		}
	}()

	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		stopConsumer()
		log.Printf("Server startup failed: %v", err)
	}
}

func startGenerationTaskConsumer(
	signalContext context.Context,
	taskService *streamdomain.GormTaskStore,
	streamService *streamdomain.Service,
	workspaceID uuid.UUID,
) (func(), error) {
	rabbitURL := os.Getenv("RABBITMQ_URL")
	if rabbitURL == "" {
		log.Println("RabbitMQ is not configured, generation consumer disabled")
		return func() {}, nil
	}

	conn, err := amqp.Dial(rabbitURL)
	if err != nil {
		return func() {}, err
	}

	channel, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return func() {}, err
	}

	exchangeName := envOrDefault("RABBITMQ_EXCHANGE", "inkwords.events")
	queueName := envOrDefault("RABBITMQ_GENERATION_QUEUE", "inkwords.generation")
	routingKey := sharedrabbitmq.GenerationRequestedMessage{}.RoutingKey()

	if err := channel.ExchangeDeclare(exchangeName, "topic", true, false, false, false, nil); err != nil {
		_ = channel.Close()
		_ = conn.Close()
		return func() {}, err
	}
	queue, err := channel.QueueDeclare(queueName, true, false, false, false, nil)
	if err != nil {
		_ = channel.Close()
		_ = conn.Close()
		return func() {}, err
	}
	if err := channel.QueueBind(queue.Name, routingKey, exchangeName, false, nil); err != nil {
		_ = channel.Close()
		_ = conn.Close()
		return func() {}, err
	}

	deliveries, err := channel.Consume(queue.Name, "llm-stream-generation-worker", false, false, false, false, nil)
	if err != nil {
		_ = channel.Close()
		_ = conn.Close()
		return func() {}, err
	}

	textbookRunner, err := textbookSampleRunnerFromEnvironment()
	if err != nil {
		return func() {}, err
	}
	consumer := streamdomain.NewTaskConsumer(taskService, streamService).WithLocalWorkspace(workspaceID).WithTextbookSampleRunner(textbookRunner)
	go func() {
		for {
			select {
			case <-signalContext.Done():
				return
			case delivery, ok := <-deliveries:
				if !ok {
					return
				}

				var message sharedrabbitmq.GenerationRequestedMessage
				if err := json.Unmarshal(delivery.Body, &message); err != nil {
					log.Printf("invalid generation message payload: %v", err)
					if ackErr := ackDelivery(delivery, "malformed generation message"); ackErr != nil {
						log.Printf("generation delivery acknowledgement failed: %v", ackErr)
					}
					continue
				}

				if err := consumer.HandleGenerationRequested(signalContext, message); err != nil {
					log.Printf("generation task handling failed for %s: %v", message.TaskID, err)
					if nackErr := nackDelivery(delivery, message.TaskID); nackErr != nil {
						log.Printf("generation delivery rejection failed: %v", nackErr)
					}
					continue
				}

				if ackErr := ackDelivery(delivery, "completed generation task "+message.TaskID.String()); ackErr != nil {
					log.Printf("generation delivery acknowledgement failed: %v", ackErr)
				}
			}
		}
	}()

	stop := func() {
		_ = channel.Close()
		_ = conn.Close()
	}

	return stop, nil
}

// textbookSampleRunnerFromEnvironment keeps real provider use opt-in. The
// default fixture runner is deterministic and network-free; an operator must
// set both a provider and the appropriate environment-injected key before any
// remote model can be selected.
func textbookSampleRunnerFromEnvironment() (*textbookapp.SampleTaskRunner, error) {
	provider := strings.ToLower(strings.TrimSpace(os.Getenv("TEXTBOOK_GENERATION_PROVIDER")))
	if provider == "" || provider == "fake" {
		return textbookapp.NewSampleTaskRunner(), nil
	}
	port, provider, model, err := textbookProviderPortFromEnvironment()
	if err != nil {
		return nil, err
	}
	generator := textbookapp.NewCachedPortSampleGenerator(port, provider, model, textbookapp.DefaultSampleGenerationBudget(), textbookapp.NewMemoryGenerationResultCache())
	if directory := strings.TrimSpace(os.Getenv("TEXTBOOK_REJECTED_DRAFTS_DIR")); directory != "" {
		store, err := rejecteddraft.Open(directory)
		if err != nil {
			return nil, fmt.Errorf("private rejected draft storage unavailable")
		}
		return textbookapp.NewSampleTaskRunner(textbookapp.NewRetainingSampleGenerator(generator, store)).WithCorrectionStore(store), nil
	}
	return textbookapp.NewSampleTaskRunner(generator), nil
}

func textbookProviderPortFromEnvironment() (sharedgeneration.Port, string, string, error) {
	provider := strings.ToLower(strings.TrimSpace(os.Getenv("TEXTBOOK_GENERATION_PROVIDER")))
	if provider == "" || provider == "fake" {
		return nil, "", "", fmt.Errorf("no external textbook generation provider is configured")
	}
	policy := textbookapp.TaskModelPolicy{
		CoreModel:     os.Getenv("TEXTBOOK_CORE_MODEL"),
		StandardModel: os.Getenv("TEXTBOOK_STANDARD_MODEL"),
	}
	model, err := policy.ModelFor(textbookapp.GenerationTaskSampleChapter)
	if err != nil {
		return nil, "", "", err
	}
	requestTimeout, err := textbookGenerationRequestTimeoutFromEnvironment()
	if err != nil {
		return nil, "", "", err
	}
	var port sharedgeneration.Port
	switch provider {
	case "deepseek":
		key := strings.TrimSpace(os.Getenv("DEEPSEEK_API_KEY"))
		if key == "" {
			return nil, "", "", fmt.Errorf("textbook DeepSeek provider key is not configured")
		}
		port = platformllm.NewDeepSeekGenerationAdapterWithTimeout(platformllm.NewDeepSeekClient(key), requestTimeout)
	case "openai":
		key := strings.TrimSpace(os.Getenv("OPENAI_API_KEY"))
		if key == "" {
			return nil, "", "", fmt.Errorf("textbook OpenAI provider key is not configured")
		}
		port = platformllm.NewOpenAIGenerationAdapterWithTimeout(key, requestTimeout)
	default:
		return nil, "", "", fmt.Errorf("unsupported textbook generation provider %q", provider)
	}
	return port, provider, model, nil
}

func textbookGenerationRequestTimeoutFromEnvironment() (time.Duration, error) {
	raw := strings.TrimSpace(os.Getenv("TEXTBOOK_GENERATION_REQUEST_TIMEOUT"))
	if raw == "" {
		return defaultTextbookGenerationRequestTimeout, nil
	}
	timeout, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("parse TEXTBOOK_GENERATION_REQUEST_TIMEOUT: %w", err)
	}
	if timeout < minTextbookGenerationRequestTimeout || timeout > maxTextbookGenerationRequestTimeout {
		return 0, fmt.Errorf("TEXTBOOK_GENERATION_REQUEST_TIMEOUT must be between %s and %s", minTextbookGenerationRequestTimeout, maxTextbookGenerationRequestTimeout)
	}
	return timeout, nil
}

type providerConnectionTesterFactory func() (*textbookapp.ProviderConnectionTester, error)

func providerConnectionTesterFromEnvironment() (*textbookapp.ProviderConnectionTester, error) {
	port, provider, model, err := textbookProviderPortFromEnvironment()
	if err != nil {
		return nil, err
	}
	return textbookapp.NewProviderConnectionTester(port, provider, model), nil
}

func providerConnectionHandler(factory providerConnectionTesterFactory) gin.HandlerFunc {
	return func(c *gin.Context) {
		tester, err := factory()
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"code": "PROVIDER_CONNECTION_UNAVAILABLE", "message": "尚未配置可测试的模型提供商", "data": nil})
			return
		}
		result, err := tester.Test(c.Request.Context())
		if err != nil {
			// Provider error bodies can contain credential-adjacent diagnostics.
			// They stay in neither HTTP responses nor durable logs.
			c.JSON(http.StatusBadGateway, gin.H{"code": "PROVIDER_CONNECTION_FAILED", "message": "模型连接测试失败；请检查本机提供商配置后重试", "data": nil})
			return
		}
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": result})
	}
}

func envOrDefault(key string, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

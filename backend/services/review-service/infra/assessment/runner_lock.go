package assessment

import (
	"context"
	"fmt"
	"time"
)

// A session lock keeps aggregate-server and review-service processes from
// treating each other's live calls as abandoned during local startup recovery.
const runnerLockID int64 = 734212902

func (s *Store) checkRunner(ctx context.Context) error {
	s.leaderMu.Lock()
	defer s.leaderMu.Unlock()
	if s.leader == nil {
		return fmt.Errorf("评分执行器尚未初始化")
	}
	var alive int
	if err := s.leader.QueryRowContext(ctx, "SELECT 1").Scan(&alive); err != nil {
		return fmt.Errorf("评分执行器连接已中断，请重启服务后恢复")
	}
	return nil
}

func (s *Store) acquireRunner(ctx context.Context) error {
	s.leaderMu.Lock()
	defer s.leaderMu.Unlock()
	if s.leader != nil {
		return nil
	}
	pool, err := s.db.DB()
	if err != nil {
		return err
	}
	connection, err := pool.Conn(ctx)
	if err != nil {
		return err
	}
	var acquired bool
	err = connection.QueryRowContext(ctx, "SELECT pg_try_advisory_lock($1)", runnerLockID).Scan(&acquired)
	if err != nil || !acquired {
		_ = connection.Close()
		return fmt.Errorf("评分执行器不可用或已在另一个进程启动")
	}
	s.leader = connection
	return nil
}

// Close releases the dedicated runner connection for tests or an orderly shutdown.
func (s *Store) Close() error {
	s.leaderMu.Lock()
	defer s.leaderMu.Unlock()
	if s.leader == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, unlockErr := s.leader.ExecContext(ctx, "SELECT pg_advisory_unlock($1)", runnerLockID)
	closeErr := s.leader.Close()
	s.leader = nil
	if unlockErr != nil {
		return unlockErr
	}
	return closeErr
}

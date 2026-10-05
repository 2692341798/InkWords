package textbook

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestPlainLanguageRisksFlagsRepeatedAcronyms(t *testing.T) {
	require.Equal(t, []string{"repeated_acronym:HTTP"}, PlainLanguageRisks("HTTP 是协议。HTTP 请求会到服务端。"))
	require.Empty(t, PlainLanguageRisks("先用白话解释，再给术语。"))
}

func TestUndefinedRepeatedAcronymsRequiresDefinitionInProse(t *testing.T) {
	require.Equal(t, []string{"HTTP"}, UndefinedRepeatedAcronyms("HTTP 请求到达服务端。HTTP 响应再返回浏览器。"))
	require.Empty(t, UndefinedRepeatedAcronyms("HTTP（超文本传输协议）规定浏览器和服务端怎样交换请求。HTTP 请求可以包含路径。"))
	require.Empty(t, UndefinedRepeatedAcronyms("```go\nHTTP HTTP HTTP\n```\n正文不依赖代码中的缩写。"))
}

func TestAcronymDefinitionAllowsMatchedInlineFormatting(t *testing.T) {
	for _, marker := range []string{"`", "**", "*"} {
		require.Empty(t, UndefinedRepeatedAcronyms(marker+"POST"+marker+"（提交方法）发送数据。POST 请求仍按方法分派。"))
		require.Equal(t, []string{"POST"}, UndefinedRepeatedAcronyms(marker+"POST"+marker+" 发送数据。POST（提交方法）后来才解释。"))
	}
	require.Equal(t, []string{"POST"}, UndefinedRepeatedAcronyms("POST`（提交方法）发送数据。POST 请求。"))
}

func TestCircularTermDefinitionsFlagsOnlyDefinitionsThatStartByRepeatingTheTerm(t *testing.T) {
	require.Equal(t, []string{"HTTP", "路由"}, CircularTermDefinitions("HTTP 是 HTTP 的简称。\n路由就是路由规则本身。"))
	require.Empty(t, CircularTermDefinitions("路由是把请求路径映射到处理函数的规则。\nHTTP（超文本传输协议）规定消息怎样交换。"))
	require.Empty(t, CircularTermDefinitions("```text\n路由是路由的规则\n```\n代码块不构成正文定义。"))
}

func TestMissingCodeBlockOriginsRequiresAdjacentSupportedLabel(t *testing.T) {
	withOrigin := "**代码来源：教学实现。**\n\n```go\npackage main\n```"
	require.Empty(t, MissingCodeBlockOrigins(withOrigin))
	require.Equal(t, []string{"code_block_1"}, MissingCodeBlockOrigins("```go\npackage main\n```"))
	require.Equal(t, []string{"code_block_1"}, MissingCodeBlockOrigins("**代码来源：其他网站。**\n\n```go\npackage main\n```"))
}

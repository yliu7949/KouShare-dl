package color

import "github.com/fatih/color"

// 参考 https://en.wikipedia.org/wiki/ANSI_escape_code 查看各个平台/软件上的颜色显示效果
var (
	Cyan   = color.New(color.FgHiCyan)
	Yellow = color.New(color.FgHiYellow)
	Red    = color.New(color.FgHiRed)
	Green  = color.New(color.FgHiGreen)
)

// DisableColor 显式决定是否使用彩色输出
func DisableColor(b bool) {
	color.NoColor = b
}

// Emphasize 使用青色强调内容
func Emphasize(s string) string {
	return Cyan.Sprint(s)
}

// Highlight 使用黄色高亮内容
func Highlight(s string) string {
	return Yellow.Sprint(s)
}

// Error 使用红色提示错误
func Error(s string) string {
	return Red.Sprint(s)
}

// Done 使用绿色提示完成
func Done(s string) string {
	return Green.Sprint(s)
}

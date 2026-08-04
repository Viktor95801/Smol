package lexer

func isAlpha(c uint8) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c == '_')
}

func isDigit(c uint8) bool {
	return (c >= '0' && c <= '9')
}

func isAlnum(c uint8) bool {
	return isAlpha(c) || isDigit(c)
}

func isSpace(c uint8) bool {
	return (c == ' ') || (c == '\t') || (c == '\n') || (c == '\r') || (c == '\v') || (c == '\f')
}

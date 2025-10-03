package stream

import (
	"go-parser/pkg/lexer"
)

type TokenStream interface {
	Current() lexer.Token
	Next() lexer.Token
	Peek() lexer.Token
	PeekN(n int) lexer.Token
	Consume() lexer.Token
	ConsumeAll() []lexer.Token
	Position() int
	SetPosition(pos int)
	HasMore() bool
	Clone() TokenStream
}

type SimpleTokenStream struct {
	lexer    lexer.Lexer
	tokens   []lexer.Token
	position int
	current  lexer.Token
}

func NewTokenStream(l lexer.Lexer) *SimpleTokenStream {
	s := &SimpleTokenStream{
		lexer:    l,
		tokens:   make([]lexer.Token, 0),
		position: 0,
	}

	// Буферизуем все токены сразу при создании потока
	for {
		token := s.lexer.NextToken()
		s.tokens = append(s.tokens, token)
		if token.Type == lexer.TokenEOF {
			break
		}
	}

	// Устанавливаем текущий токен
	if len(s.tokens) > 0 {
		s.current = s.tokens[0]
	}

	// Лексер больше не нужен, так как все токены буферизованы
	s.lexer = nil

	return s
}

func (s *SimpleTokenStream) Current() lexer.Token {
	return s.current
}

func (s *SimpleTokenStream) Next() lexer.Token {
	s.position++ // Увеличиваем позицию сначала

	if s.position < len(s.tokens) {
		s.current = s.tokens[s.position]
	} else if s.lexer != nil {
		// Если лексер доступен, получаем новый токен
		s.current = s.lexer.NextToken()
		s.tokens = append(s.tokens, s.current)
	} else {
		// Если лексера нет (в клоне), возвращаем EOF
		s.current = lexer.Token{Type: lexer.TokenEOF}
	}
	return s.current
}

func (s *SimpleTokenStream) Peek() lexer.Token {
	return s.PeekN(1)
}

func (s *SimpleTokenStream) PeekN(n int) lexer.Token {
	targetPos := s.position + n

	// Загружаем токены до нужной позиции
	for len(s.tokens) <= targetPos {
		if s.lexer != nil {
			token := s.lexer.NextToken()
			s.tokens = append(s.tokens, token)
			if token.Type == lexer.TokenEOF {
				break
			}
		} else {
			// Если лексера нет, больше нет токенов для загрузки
			break
		}
	}

	if targetPos < len(s.tokens) {
		return s.tokens[targetPos]
	}
	return lexer.Token{Type: lexer.TokenEOF}
}

func (s *SimpleTokenStream) Consume() lexer.Token {
	token := s.Current()
	s.Next()
	return token
}

func (s *SimpleTokenStream) ConsumeAll() []lexer.Token {
	var result []lexer.Token
	for s.HasMore() {
		result = append(result, s.Consume())
	}
	return result
}

func (s *SimpleTokenStream) Position() int {
	return s.position
}

func (s *SimpleTokenStream) SetPosition(pos int) {
	if pos < 0 {
		return // Неверная позиция
	}

	if pos == 0 {
		// Восстановление на начало потока
		s.position = 0
		// Если есть токены в буфере, берем первый
		if len(s.tokens) > 0 {
			s.current = s.tokens[0]
		} else {
			// Иначе просто устанавливаем позицию 0, current останется неизменным
			// Это не идеальное решение, но лучше чем ничего
			s.position = 0
		}
	} else if pos < len(s.tokens) {
		// Восстановление на существующую позицию
		s.position = pos
		s.current = s.tokens[pos]
	}
	// Если pos >= len(s.tokens), ничего не делаем (остаемся в текущей позиции)
}

func (s *SimpleTokenStream) HasMore() bool {
	return s.current.Type != lexer.TokenEOF
}

func (s *SimpleTokenStream) Clone() TokenStream {
	// Для клонирования копируем только уже буферизованные токены
	// НЕ потребляем токены из лексера, чтобы не нарушать состояние оригинала
	allTokens := make([]lexer.Token, len(s.tokens))
	copy(allTokens, s.tokens)

	// Создаем клон с копией токенов и позиции
	clone := &SimpleTokenStream{
		lexer:    nil, // Клон не должен иметь доступа к лексеру
		tokens:   allTokens,
		position: s.position,
		current:  s.current,
	}

	return clone
}

package ram

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

const (
	OcNone OpCode = iota
	OcRead
	OcWrite
	OcLoad
	OcStore
	OcAdd
	OcSub
	OcMul
	OcDiv
	OcJump
	OcJgtz
	OcJzero
	OcHalt
)

type OpCode byte

type Program struct {
	commands []Command
}

func NewProgram(r io.Reader) (*Program, error) {
	var commands []Command
	parser := newParser(r)
	for {
		cmd, err := parser.parseCommand()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		}
		commands = append(commands, cmd)
	}
	if len(commands) == 0 {
		return nil, errors.New("no command found")
	}
	return &Program{commands: commands}, nil
}

func (p *Program) Len() int {
	return len(p.commands)
}

func (p *Program) Get(i int) *Command {
	if i < 0 || i >= len(p.commands) {
		return nil
	}
	return &p.commands[i]
}

type Command struct {
	Code OpCode
	Op   Operand
}

const (
	AmNone AccessMode = 0
	AmImm  AccessMode = 1 << (iota - 1)
	AmReg
	AmPtr
)

type AccessMode byte

type Operand struct {
	Mode AccessMode
	Val  int
}

func (op Operand) String() string {
	return fmt.Sprintf("%d:%d", op.Mode, op.Val)
}

type parser struct {
	scanner *bufio.Scanner
}

func newParser(r io.Reader) *parser {
	scanner := bufio.NewScanner(r)
	scanner.Split(bufio.ScanWords)
	return &parser{scanner: scanner}
}

func (p *parser) parseCommand() (Command, error) {
	if !p.scanner.Scan() {
		if err := p.scanner.Err(); err != nil {
			return Command{}, err
		}
		return Command{}, io.EOF
	}

	text := strings.ToUpper(p.scanner.Text())

	cdef, ok := commandDefs[text]
	if !ok {
		return Command{}, fmt.Errorf("unknown command: %s", text)
	}
	if cdef.supportedModes == AmNone {
		return Command{Code: cdef.code}, nil
	}

	op, err := p.parseOperand()
	if err != nil {
		return Command{}, err
	}
	if !cdef.supports(op.Mode) {
		return Command{}, fmt.Errorf("unexpected operand type: %v", op.Mode)
	}

	return Command{Code: cdef.code, Op: op}, nil
}

var errExpectedOperand = errors.New("expected operand")

func (p *parser) parseOperand() (Operand, error) {
	if !p.scanner.Scan() {
		if err := p.scanner.Err(); err != nil {
			return Operand{}, err
		}
		return Operand{}, errExpectedOperand
	}

	text := p.scanner.Text()
	if len(text) == 0 {
		return Operand{}, errExpectedOperand
	}

	mode := AmReg
	switch text[0] {
	case '=':
		mode = AmImm
		text = text[1:]
	case '*':
		mode = AmPtr
		text = text[1:]
	}
	val, err := strconv.Atoi(text)
	if err != nil {
		return Operand{}, err
	}
	if (mode == AmReg || mode == AmPtr) && val <= 0 {
		return Operand{}, fmt.Errorf("invalid value for operand: %s", text)
	}

	return Operand{Mode: mode, Val: val}, nil
}

var commandDefs = map[string]commandDef{
	"READ":  {OcRead, AmReg | AmPtr},
	"WRITE": {OcWrite, AmImm | AmReg | AmPtr},
	"LOAD":  {OcLoad, AmImm | AmReg | AmPtr},
	"STORE": {OcStore, AmReg | AmPtr},
	"ADD":   {OcAdd, AmImm | AmReg | AmPtr},
	"SUB":   {OcSub, AmImm | AmReg | AmPtr},
	"MUL":   {OcMul, AmImm | AmReg | AmPtr},
	"DIV":   {OcDiv, AmImm | AmReg | AmPtr},
	"JUMP":  {OcJump, AmImm | AmReg | AmPtr},
	"JGTZ":  {OcJgtz, AmImm | AmReg | AmPtr},
	"JZERO": {OcJzero, AmImm | AmReg | AmPtr},
	"HALT":  {OcHalt, AmNone},
}

type commandDef struct {
	code           OpCode
	supportedModes AccessMode
}

func (c *commandDef) supports(m AccessMode) bool {
	return c.supportedModes&m != 0
}

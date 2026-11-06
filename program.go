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
	labelToCmd := make(map[string]int)
	parser := newParser(r)
	for {
		cmd, err := parser.parseCommand()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		}
		if len(cmd.Label) > 0 {
			if _, found := labelToCmd[cmd.Label]; found {
				return nil, fmt.Errorf("duplicate label: %s", cmd.Label)
			}
			labelToCmd[cmd.Label] = len(commands)
		}
		commands = append(commands, cmd)
	}
	if len(commands) == 0 {
		return nil, errors.New("no command found")
	}

	program := &Program{commands: commands}

	return program.resolveTargetLabels(labelToCmd)
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

func (p *Program) resolveTargetLabels(labelToCmd map[string]int) (*Program, error) {
	for i := range p.commands {
		cmd := &p.commands[i]
		targetLabel := cmd.Op.TargetLabel
		if len(targetLabel) > 0 {
			targetIndex, ok := labelToCmd[targetLabel]
			if !ok {
				return nil, fmt.Errorf("unresolved target label: %s", targetLabel)
			}
			cmd.Op.Val = targetIndex
		}
	}
	return p, nil
}

type Command struct {
	Label string
	Code  OpCode
	Op    Operand
}

const (
	AmNone AccessMode = 0
	AmImm  AccessMode = 1 << (iota - 1)
	AmReg
	AmPtr
)

type AccessMode byte

type Operand struct {
	Mode        AccessMode
	Val         int
	TargetLabel string
}

func (op *Operand) String() string {
	if len(op.TargetLabel) > 0 {
		return "#" + op.TargetLabel
	}
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

var errEmptyLabel = errors.New("empty label")

func (p *parser) parseCommand() (Command, error) {
	labelOrCode, err := p.nextToken()
	if err != nil {
		return Command{}, err
	}

	label := ""
	code := labelOrCode
	if strings.HasSuffix(labelOrCode, ":") {
		label = labelOrCode[:len(labelOrCode)-1]
		if len(label) == 0 {
			return Command{}, errEmptyLabel
		}
		code, err = p.nextToken()
		if err != nil {
			return Command{}, err
		}
	}

	code = strings.ToUpper(code)

	cdef, ok := commandDefs[code]
	if !ok {
		return Command{}, fmt.Errorf("unknown command: %s", code)
	}
	if cdef.supportedModes == AmNone {
		return Command{Label: label, Code: cdef.code}, nil
	}

	op, err := p.parseOperand()
	if err != nil {
		return Command{}, err
	}

	if !cdef.supports(op.Mode) {
		return Command{}, fmt.Errorf("unexpected access mode: %d", op.Mode)
	}

	if len(op.TargetLabel) > 0 && !cdef.canHaveLabel {
		return Command{}, fmt.Errorf("unexpected target label: %s", op.TargetLabel)
	}

	return Command{Label: label, Code: cdef.code, Op: op}, nil
}

var errExpectedOperand = errors.New("expected operand")
var errEmptyTargetLabel = errors.New("empty label")

func (p *parser) parseOperand() (Operand, error) {
	token, err := p.nextToken()
	if err != nil {
		if errors.Is(err, io.EOF) {
			err = errExpectedOperand
		}
		return Operand{}, err
	}

	if len(token) == 0 {
		return Operand{}, errExpectedOperand
	}

	mode := AmReg
	targetLabel := ""
	switch token[0] {
	case '=':
		mode = AmImm
		token = token[1:]
	case '*':
		mode = AmPtr
		token = token[1:]
	case '#':
		mode = AmImm
		targetLabel = token[1:]
		if len(targetLabel) == 0 {
			return Operand{}, errEmptyTargetLabel
		}
		return Operand{Mode: mode, TargetLabel: targetLabel}, nil
	}

	val, err := strconv.Atoi(token)
	if err != nil {
		return Operand{}, err
	}
	if (mode == AmReg || mode == AmPtr) && val <= 0 {
		return Operand{}, fmt.Errorf("expected integer: %s", token)
	}

	return Operand{Mode: mode, Val: val}, nil
}

func (p *parser) nextToken() (string, error) {
	if !p.scanner.Scan() {
		if err := p.scanner.Err(); err != nil {
			return "", err
		}
		return "", io.EOF
	}
	return p.scanner.Text(), nil
}

var commandDefs = map[string]commandDef{
	"READ":  {OcRead, AmReg | AmPtr, false},
	"WRITE": {OcWrite, AmImm | AmReg | AmPtr, false},
	"LOAD":  {OcLoad, AmImm | AmReg | AmPtr, false},
	"STORE": {OcStore, AmReg | AmPtr, false},
	"ADD":   {OcAdd, AmImm | AmReg | AmPtr, false},
	"SUB":   {OcSub, AmImm | AmReg | AmPtr, false},
	"MUL":   {OcMul, AmImm | AmReg | AmPtr, false},
	"DIV":   {OcDiv, AmImm | AmReg | AmPtr, false},
	"JUMP":  {OcJump, AmImm | AmReg | AmPtr, true},
	"JGTZ":  {OcJgtz, AmImm | AmReg | AmPtr, true},
	"JZERO": {OcJzero, AmImm | AmReg | AmPtr, true},
	"HALT":  {OcHalt, AmNone, false},
}

type commandDef struct {
	code           OpCode
	supportedModes AccessMode
	canHaveLabel   bool
}

func (c *commandDef) supports(m AccessMode) bool {
	return c.supportedModes&m != 0
}

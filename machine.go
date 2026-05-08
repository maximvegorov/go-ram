package ram

import (
	"errors"
	"fmt"
)

var (
	ErrHalt      = errors.New("halt")
	ErrDivByZero = errors.New("div by zero")
)

type Machine struct {
	program *Program
	input   Input
	output  Output
	regs    registers
	ip      int
	err     error
}

func NewMachine(program *Program, input Input, output Output) *Machine {
	return &Machine{program: program, input: input, output: output, regs: registers{}}
}

func (m *Machine) Err() error {
	return m.err
}

func (m *Machine) ExecuteAll() error {
	for {
		err := m.ExecuteStep()
		if err != nil {
			return err
		}
	}
}

func (m *Machine) ExecuteStep() error {
	if m.err != nil {
		return m.err
	}
	nextCommand, err := m.doExecuteStep()
	if err != nil {
		m.err = err
		return err
	}
	m.ip = nextCommand
	return nil
}

func (m *Machine) doExecuteStep() (nip int, execErr error) {
	cmd := m.program.Get(m.ip)
	if cmd == nil {
		return 0, fmt.Errorf("invalid ip: %d", m.ip)
	}
	switch cmd.Code {
	case OcRead:
		reg, err := m.getTargetReg(cmd.Op)
		if err != nil {
			return 0, err
		}
		val, err := m.input.Read()
		if err != nil {
			return 0, err
		}
		m.regs.set(reg, val)
	case OcWrite:
		val, err := m.getValue(cmd.Op)
		if err != nil {
			return 0, err
		}
		if err := m.output.Write(val); err != nil {
			return 0, err
		}
	case OcLoad:
		val, err := m.getValue(cmd.Op)
		if err != nil {
			return 0, err
		}
		m.regs.set(0, val)
	case OcStore:
		reg, err := m.getTargetReg(cmd.Op)
		if err != nil {
			return 0, err
		}
		m.regs.set(reg, m.regs.get(0))
	case OcAdd:
		val, err := m.getValue(cmd.Op)
		if err != nil {
			return 0, err
		}
		m.regs.set(0, m.regs.get(0)+val)
	case OcSub:
		val, err := m.getValue(cmd.Op)
		if err != nil {
			return 0, err
		}
		m.regs.set(0, m.regs.get(0)-val)
	case OcMul:
		val, err := m.getValue(cmd.Op)
		if err != nil {
			return 0, err
		}
		m.regs.set(0, m.regs.get(0)*val)
	case OcDiv:
		val, err := m.getValue(cmd.Op)
		if err != nil {
			return 0, err
		}
		if val == 0 {
			return 0, ErrDivByZero
		}
		m.regs.set(0, m.regs.get(0)/val)
	case OcJump:
		target, err := m.getTargetLabel(cmd.Op)
		if err != nil {
			return 0, err
		}
		return target, nil
	case OcJgtz:
		target, err := m.getTargetLabel(cmd.Op)
		if err != nil {
			return 0, err
		}
		if m.regs.get(0) > 0 {
			return target, nil
		}
	case OcJzero:
		target, err := m.getTargetLabel(cmd.Op)
		if err != nil {
			return 0, err
		}
		if m.regs.get(0) == 0 {
			return target, nil
		}
	case OcHalt:
		return 0, ErrHalt

	default:
		return 0, fmt.Errorf("unknown command: %d", cmd.Code)
	}
	return m.ip + 1, nil
}

func (m *Machine) getValue(op Operand) (int, error) {
	if op.Mode == AmImm {
		return op.Val, nil
	}
	reg, err := m.getTargetReg(op)
	if err != nil {
		return 0, err
	}
	return m.regs.get(reg), nil
}

func (m *Machine) getTargetReg(op Operand) (int, error) {
	reg := op.Val
	if reg > 0 && op.Mode == AmPtr {
		reg = m.regs.get(reg)
	}
	if reg <= 0 {
		return 0, fmt.Errorf("invalid register %v", op)
	}
	return reg, nil
}

func (m *Machine) getTargetLabel(op Operand) (int, error) {
	val, err := m.getValue(op)
	if err != nil {
		return 0, err
	}
	if val < 0 || val >= m.program.Len() {
		return 0, fmt.Errorf("invalid label %v", op)
	}
	return val, nil
}

type registers struct {
	values []int
}

func (regs *registers) get(i int) int {
	if i >= len(regs.values) {
		return 0
	}
	return regs.values[i]
}

func (regs *registers) set(i int, val int) {
	if i >= len(regs.values) {
		values := make([]int, max(2*len(regs.values), i+1))
		copy(values, regs.values)
		regs.values = values
	}
	regs.values[i] = val
}

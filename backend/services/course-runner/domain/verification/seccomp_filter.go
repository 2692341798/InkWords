package verification

import (
	"encoding/binary"
	"fmt"
)

const (
	bpfLoadWordAbsolute = 0x20
	bpfJumpEqual        = 0x15
	bpfJumpSet          = 0x45
	bpfReturn           = 0x06

	seccompReturnKillProcess = 0x80000000
	seccompReturnErrno       = 0x00050000
	seccompReturnAllow       = 0x7fff0000
	linuxEPERM               = 1
	linuxENOSYS              = 38
	cloneNamespaceMask       = 0x7e020080
)

type sandboxLinuxSyscalls struct {
	auditArchitecture uint32
	clone             uint32
	clone3            uint32
	mount             uint32
	pivotRoot         uint32
	setns             uint32
	umount2           uint32
	unshare           uint32
}

type classicBPFInstruction struct {
	code uint16
	jt   uint8
	jf   uint8
	k    uint32
}

// sandboxSeccompBPF restores the namespace and mount restrictions relaxed only
// long enough for Bubblewrap to construct the inner sandbox. The filter is
// installed by Bubblewrap immediately before it executes untrusted content.
func sandboxSeccompBPF(architecture string) ([]byte, error) {
	var calls sandboxLinuxSyscalls
	switch architecture {
	case "arm64":
		calls = sandboxLinuxSyscalls{auditArchitecture: 0xc00000b7, clone: 220, clone3: 435, mount: 40, pivotRoot: 41, setns: 268, umount2: 39, unshare: 97}
	case "amd64":
		calls = sandboxLinuxSyscalls{auditArchitecture: 0xc000003e, clone: 56, clone3: 435, mount: 165, pivotRoot: 155, setns: 308, umount2: 166, unshare: 272}
	default:
		return nil, fmt.Errorf("unsupported learner sandbox architecture %q", architecture)
	}

	instructions := []classicBPFInstruction{
		{code: bpfLoadWordAbsolute, k: 4},
		{code: bpfJumpEqual, jt: 1, k: calls.auditArchitecture},
		{code: bpfReturn, k: seccompReturnKillProcess},
		{code: bpfLoadWordAbsolute, k: 0},
	}
	for _, number := range []uint32{calls.mount, calls.pivotRoot, calls.setns, calls.umount2, calls.unshare} {
		instructions = append(instructions,
			classicBPFInstruction{code: bpfJumpEqual, jf: 1, k: number},
			classicBPFInstruction{code: bpfReturn, k: seccompReturnErrno | linuxEPERM},
		)
	}
	// clone3 passes flags through a pointer that classic BPF cannot inspect.
	// ENOSYS makes Go use clone, whose namespace bits are inspected below.
	instructions = append(instructions,
		classicBPFInstruction{code: bpfJumpEqual, jf: 1, k: calls.clone3},
		classicBPFInstruction{code: bpfReturn, k: seccompReturnErrno | linuxENOSYS},
		classicBPFInstruction{code: bpfJumpEqual, jf: 3, k: calls.clone},
		classicBPFInstruction{code: bpfLoadWordAbsolute, k: 16},
		classicBPFInstruction{code: bpfJumpSet, jf: 1, k: cloneNamespaceMask},
		classicBPFInstruction{code: bpfReturn, k: seccompReturnErrno | linuxEPERM},
		classicBPFInstruction{code: bpfReturn, k: seccompReturnAllow},
	)

	encoded := make([]byte, len(instructions)*8)
	for index, instruction := range instructions {
		offset := index * 8
		binary.LittleEndian.PutUint16(encoded[offset:offset+2], instruction.code)
		encoded[offset+2], encoded[offset+3] = instruction.jt, instruction.jf
		binary.LittleEndian.PutUint32(encoded[offset+4:offset+8], instruction.k)
	}
	return encoded, nil
}

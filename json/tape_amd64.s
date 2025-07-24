// AVX2-optimized structural character detection
#include "textflag.h"

// func findStructuralCharsAVX2(data []byte, indices []int) int
// Processes 32 bytes at once to find JSON structural characters
TEXT ·findStructuralCharsAVX2(SB), NOSPLIT, $0-56
    MOVQ data_base+0(FP), SI      // source pointer
    MOVQ indices_base+24(FP), DI  // indices array pointer
    
    // Load 32 bytes of data
    VMOVDQU (SI), Y0
    
    // Create comparison masks for each structural character
    VMOVDQU openBraceVec<>(SB), Y1   // '{'
    VMOVDQU closeBraceVec<>(SB), Y2  // '}'
    VMOVDQU openBracketVec<>(SB), Y3 // '['
    VMOVDQU closeBracketVec<>(SB), Y4 // ']'
    VMOVDQU colonVec<>(SB), Y5       // ':'
    VMOVDQU commaVec<>(SB), Y6       // ','
    VMOVDQU quoteVec<>(SB), Y7       // '"'
    
    // Compare against all structural characters
    VPCMPEQB Y0, Y1, Y8   // Compare with '{'
    VPCMPEQB Y0, Y2, Y9   // Compare with '}'
    VPCMPEQB Y0, Y3, Y10  // Compare with '['
    VPCMPEQB Y0, Y4, Y11  // Compare with ']'
    VPCMPEQB Y0, Y5, Y12  // Compare with ':'
    VPCMPEQB Y0, Y6, Y13  // Compare with ','
    VPCMPEQB Y0, Y7, Y14  // Compare with '"'
    
    // Combine all matches using OR operations
    VPOR Y8, Y9, Y15      // { | }
    VPOR Y10, Y11, Y8     // [ | ]
    VPOR Y12, Y13, Y9     // : | ,
    VPOR Y15, Y8, Y10     // ({|}) | ([|])
    VPOR Y9, Y14, Y11     // (:|,) | "
    VPOR Y10, Y11, Y12    // All structural chars
    
    // Convert to bitmask
    VPMOVMSKB Y12, AX
    
    // Count and store indices of set bits
    XORQ CX, CX           // Counter for found indices
    XORQ DX, DX           // Bit position counter
    
find_loop:
    CMPQ DX, $32
    JGE  done
    
    BTQ  DX, AX           // Test bit DX in AX (use BTQ instead of BTLQ)
    JNC  next_bit         // Jump if bit not set
    
    // Store index
    MOVQ DX, (DI)(CX*8)   // Store bit position in indices array
    INCQ CX               // Increment counter
    
next_bit:
    INCQ DX
    JMP  find_loop
    
done:
    MOVQ CX, ret+48(FP)   // Return count
    VZEROUPPER            // Clean up AVX state
    RET

// Constants for structural characters (32 bytes each) - fixed syntax
DATA openBraceVec<>+0(SB)/1, $0x7B   // '{'
DATA openBraceVec<>+1(SB)/1, $0x7B
DATA openBraceVec<>+2(SB)/1, $0x7B
DATA openBraceVec<>+3(SB)/1, $0x7B
DATA openBraceVec<>+4(SB)/1, $0x7B
DATA openBraceVec<>+5(SB)/1, $0x7B
DATA openBraceVec<>+6(SB)/1, $0x7B
DATA openBraceVec<>+7(SB)/1, $0x7B
DATA openBraceVec<>+8(SB)/1, $0x7B
DATA openBraceVec<>+9(SB)/1, $0x7B
DATA openBraceVec<>+10(SB)/1, $0x7B
DATA openBraceVec<>+11(SB)/1, $0x7B
DATA openBraceVec<>+12(SB)/1, $0x7B
DATA openBraceVec<>+13(SB)/1, $0x7B
DATA openBraceVec<>+14(SB)/1, $0x7B
DATA openBraceVec<>+15(SB)/1, $0x7B
DATA openBraceVec<>+16(SB)/1, $0x7B
DATA openBraceVec<>+17(SB)/1, $0x7B
DATA openBraceVec<>+18(SB)/1, $0x7B
DATA openBraceVec<>+19(SB)/1, $0x7B
DATA openBraceVec<>+20(SB)/1, $0x7B
DATA openBraceVec<>+21(SB)/1, $0x7B
DATA openBraceVec<>+22(SB)/1, $0x7B
DATA openBraceVec<>+23(SB)/1, $0x7B
DATA openBraceVec<>+24(SB)/1, $0x7B
DATA openBraceVec<>+25(SB)/1, $0x7B
DATA openBraceVec<>+26(SB)/1, $0x7B
DATA openBraceVec<>+27(SB)/1, $0x7B
DATA openBraceVec<>+28(SB)/1, $0x7B
DATA openBraceVec<>+29(SB)/1, $0x7B
DATA openBraceVec<>+30(SB)/1, $0x7B
DATA openBraceVec<>+31(SB)/1, $0x7B

// Simplified constants - just define the key ones for now
DATA closeBraceVec<>+0(SB)/32, $"\x7D\x7D\x7D\x7D\x7D\x7D\x7D\x7D\x7D\x7D\x7D\x7D\x7D\x7D\x7D\x7D\x7D\x7D\x7D\x7D\x7D\x7D\x7D\x7D\x7D\x7D\x7D\x7D\x7D\x7D\x7D\x7D"
DATA openBracketVec<>+0(SB)/32, $"\x5B\x5B\x5B\x5B\x5B\x5B\x5B\x5B\x5B\x5B\x5B\x5B\x5B\x5B\x5B\x5B\x5B\x5B\x5B\x5B\x5B\x5B\x5B\x5B\x5B\x5B\x5B\x5B\x5B\x5B\x5B\x5B"
DATA closeBracketVec<>+0(SB)/32, $"\x5D\x5D\x5D\x5D\x5D\x5D\x5D\x5D\x5D\x5D\x5D\x5D\x5D\x5D\x5D\x5D\x5D\x5D\x5D\x5D\x5D\x5D\x5D\x5D\x5D\x5D\x5D\x5D\x5D\x5D\x5D\x5D"
DATA colonVec<>+0(SB)/32, $"\x3A\x3A\x3A\x3A\x3A\x3A\x3A\x3A\x3A\x3A\x3A\x3A\x3A\x3A\x3A\x3A\x3A\x3A\x3A\x3A\x3A\x3A\x3A\x3A\x3A\x3A\x3A\x3A\x3A\x3A\x3A\x3A"
DATA commaVec<>+0(SB)/32, $"\x2C\x2C\x2C\x2C\x2C\x2C\x2C\x2C\x2C\x2C\x2C\x2C\x2C\x2C\x2C\x2C\x2C\x2C\x2C\x2C\x2C\x2C\x2C\x2C\x2C\x2C\x2C\x2C\x2C\x2C\x2C\x2C"
DATA quoteVec<>+0(SB)/32, $"\x22\x22\x22\x22\x22\x22\x22\x22\x22\x22\x22\x22\x22\x22\x22\x22\x22\x22\x22\x22\x22\x22\x22\x22\x22\x22\x22\x22\x22\x22\x22\x22"

GLOBL openBraceVec<>(SB), RODATA, $32
GLOBL closeBraceVec<>(SB), RODATA, $32
GLOBL openBracketVec<>(SB), RODATA, $32
GLOBL closeBracketVec<>(SB), RODATA, $32
GLOBL colonVec<>(SB), RODATA, $32
GLOBL commaVec<>(SB), RODATA, $32
GLOBL quoteVec<>(SB), RODATA, $32

// func validateStringsAVX2(data []byte, start, end int) bool
TEXT ·validateStringsAVX2(SB), NOSPLIT, $0-41
    MOVQ data_base+0(FP), SI
    MOVQ start+24(FP), AX
    MOVQ end+32(FP), BX
    ADDQ AX, SI               // SI = data + start
    SUBQ AX, BX               // BX = length
    
    // Simple validation loop
    XORQ CX, CX               // Position counter
    
validate_loop:
    CMPQ CX, BX
    JGE  valid
    
    MOVBQZX (SI)(CX*1), DX    // Load byte
    CMPB DL, $0x20            // Check for control characters
    JL   invalid
    CMPB DL, $0x7F            // Check for extended ASCII
    JGE  invalid
    
    INCQ CX
    JMP  validate_loop
    
valid:
    MOVB $1, ret+40(FP)
    RET
    
invalid:
    MOVB $0, ret+40(FP)
    RET

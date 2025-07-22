#include "textflag.h"

// toLower16Bytes converts 16 bytes to lowercase using SIMD
// func toLower16Bytes(data []byte)
TEXT ·toLower16Bytes(SB), NOSPLIT, $0-24
    MOVQ data_base+0(FP), AX    // Load data pointer
    MOVQ data_len+8(FP), CX     // Load length
    
    // Check if we have at least 16 bytes
    CMPQ CX, $16
    JL   done
    
    // Load 16 bytes into XMM0
    MOVOU (AX), X0
    
    // Create mask for 'A' (0x41) - subtract 1 for comparison
    MOVQ $0x4040404040404040, BX  // 'A' - 1 = 0x40
    MOVQ BX, X1
    MOVQ BX, X2
    PUNPCKLQDQ X2, X1
    
    // Create mask for 'Z' + 1 (0x5B) for upper bound
    MOVQ $0x5B5B5B5B5B5B5B5B, BX  // 'Z' + 1 = 0x5B
    MOVQ BX, X2
    MOVQ BX, X3
    PUNPCKLQDQ X3, X2
    
    // Compare: is character > ('A' - 1)?
    MOVOU X0, X3
    PCMPGTB X1, X3  // X3 = mask where data > 0x40
    
    // Compare: is ('Z' + 1) > character?
    MOVOU X2, X4
    PCMPGTB X0, X4  // X4 = mask where 0x5B > data
    
    // Combine masks: uppercase letters = (data > 0x40) AND (0x5B > data)
    PAND X4, X3
    
    // Create lowercase conversion (add 0x20)
    MOVQ $0x2020202020202020, BX
    MOVQ BX, X4
    MOVQ BX, X5
    PUNPCKLQDQ X5, X4
    
    // Apply conversion only to uppercase letters
    PAND X3, X4
    PADDB X4, X0
    
    // Store result
    MOVOU X0, (AX)
    
done:
    RET

// equalsSIMD compares two byte slices using SIMD
// func equalsSIMD(a, b []byte) bool
TEXT ·equalsSIMD(SB), NOSPLIT, $0-49
    MOVQ a_base+0(FP), AX
    MOVQ a_len+8(FP), CX
    MOVQ b_base+24(FP), BX
    MOVQ b_len+32(FP), DX
    
    // Lengths should already be equal (checked in Go)
    CMPQ CX, DX
    JNE  not_equal
    
    // For small slices, let Go handle it
    CMPQ CX, $16
    JL   fallback
    
    // Load and compare 16 bytes
    MOVOU (AX), X0
    MOVOU (BX), X1
    PCMPEQB X1, X0
    
    // Check if all bytes matched
    PMOVMSKB X0, AX
    CMPL AX, $0xFFFF
    JE   equal
    
not_equal:
    MOVB $0, ret+48(FP)
    RET
    
equal:
    MOVB $1, ret+48(FP)
    RET
    
fallback:
    MOVB $0, ret+48(FP)
    RET

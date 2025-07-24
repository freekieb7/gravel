// SIMD-optimized character scanning for JSON
#include "textflag.h"

// func scanJSONChars16(data []byte) int
// Scans 16 bytes for JSON special characters (control chars, quotes, backslashes, non-ASCII)
TEXT ·scanJSONChars16(SB), NOSPLIT, $0-32
    MOVQ data_base+0(FP), SI   // source pointer
    
    // Load 16 bytes into XMM0
    MOVOU (SI), X0
    
    // Create comparison vectors
    PXOR X1, X1               // X1 = 0x00 (for control char comparison)
    MOVQ $0x2020202020202020, AX
    MOVQ AX, X2
    PUNPCKLQDQ X2, X2         // X2 = 0x20 repeated (space char)
    
    MOVQ $0x2222222222222222, AX  
    MOVQ AX, X3
    PUNPCKLQDQ X3, X3         // X3 = 0x22 repeated (quote char)
    
    MOVQ $0x5C5C5C5C5C5C5C5C, AX
    MOVQ AX, X4  
    PUNPCKLQDQ X4, X4         // X4 = 0x5C repeated (backslash)
    
    MOVQ $0x7F7F7F7F7F7F7F7F, AX
    MOVQ AX, X5
    PUNPCKLQDQ X5, X5         // X5 = 0x7F repeated (DEL char)
    
    // Check for control characters (< 0x20)
    PCMPGTB X0, X2            // X2 = mask where data[i] < 0x20
    
    // Check for quotes (== 0x22)  
    PCMPEQB X0, X3            // X3 = mask where data[i] == 0x22
    
    // Check for backslashes (== 0x5C)
    PCMPEQB X0, X4            // X4 = mask where data[i] == 0x5C
    
    // Check for high bytes (>= 0x80) by comparing with 0x7F
    PCMPGTB X5, X0            // X0 = mask where data[i] > 0x7F
    
    // Combine all masks
    POR X3, X2                // X2 |= X3 (control chars | quotes)
    POR X4, X2                // X2 |= X4 (... | backslashes) 
    POR X0, X2                // X2 |= X0 (... | high bytes)
    
    // Convert to bitmask and find first set bit
    PMOVMSKB X2, AX           // Extract mask to AX
    
    // Find position of first set bit (first special character)
    BSFQ AX, BX               // Bit scan forward
    JZ   no_special_chars     // Jump if no bits set
    
    MOVQ BX, ret+24(FP)       // Return position
    RET

no_special_chars:
    MOVQ $16, ret+24(FP)      // Return 16 (no special chars found)
    RET

// func scanQuotesAndEscapes16(data []byte) int  
// Scans 16 bytes for quotes and escape characters only
TEXT ·scanQuotesAndEscapes16(SB), NOSPLIT, $0-32
    MOVQ data_base+0(FP), SI   // source pointer
    
    // Load 16 bytes
    MOVOU (SI), X0
    
    // Create comparison vectors for quotes and backslashes
    MOVQ $0x2222222222222222, AX
    MOVQ AX, X1
    PUNPCKLQDQ X1, X1         // X1 = 0x22 repeated (quotes)
    
    MOVQ $0x5C5C5C5C5C5C5C5C, AX  
    MOVQ AX, X2
    PUNPCKLQDQ X2, X2         // X2 = 0x5C repeated (backslashes)
    
    // Compare for quotes and backslashes
    PCMPEQB X0, X1            // X1 = mask for quotes
    PCMPEQB X0, X2            // X2 = mask for backslashes
    
    // Combine masks
    POR X2, X1                // X1 = quotes | backslashes
    
    // Convert to bitmask and find first set bit
    PMOVMSKB X1, AX
    BSFQ AX, BX
    JZ   no_quotes_escapes
    
    MOVQ BX, ret+24(FP)
    RET

no_quotes_escapes:
    MOVQ $16, ret+24(FP)
    RET

// Assembly-optimized string escaping for AMD64
// Scans string for characters that need JSON escaping

#include "textflag.h"

// func escapeStringASM(src []byte, dst []byte) (needsEscape bool, pos int)
TEXT ·escapeStringASM(SB), NOSPLIT, $0-64
    MOVQ src_base+0(FP), SI    // source pointer
    MOVQ src_len+8(FP), CX     // source length
    MOVQ dst_base+24(FP), DI   // destination pointer
    XORQ AX, AX                // position counter
    XORQ DX, DX                // needs escape flag

loop:
    CMPQ AX, CX
    JGE  done
    
    MOVBQZX (SI)(AX*1), BX     // load byte
    
    // Check for characters that need escaping
    CMPB BL, $0x20             // < 0x20 (control chars)
    JL   escape_needed
    CMPB BL, $0x22             // '"'
    JE   escape_needed
    CMPB BL, $0x5C             // '\'
    JE   escape_needed
    CMPB BL, $0x7F             // >= 0x80 (non-ASCII)
    JGE  escape_needed
    
    // Regular character, copy to destination
    MOVB BL, (DI)(AX*1)
    INCQ AX
    JMP  loop

escape_needed:
    MOVQ $1, DX                // set needs escape flag
    MOVB DL, needsEscape+48(FP)
    MOVQ AX, pos+56(FP)
    RET

done:
    MOVB DL, needsEscape+48(FP)
    MOVQ AX, pos+56(FP)
    RET

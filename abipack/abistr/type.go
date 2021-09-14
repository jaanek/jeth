package abistr

import "github.com/ledgerwatch/erigon/accounts/abi"

// requireLengthPrefix returns whether the type requires any sort of length
// prefixing.
func RequiresLengthPrefix(t abi.Type) bool {
	return t.T == abi.StringTy || t.T == abi.BytesTy || t.T == abi.SliceTy
}

// isDynamicType returns true if the type is dynamic.
// The following types are called “dynamic”:
// * bytes
// * string
// * T[] for any T
// * T[k] for any dynamic T and any k >= 0
// * (T1,...,Tk) if Ti is dynamic for some 1 <= i <= k
func IsDynamicType(t abi.Type) bool {
	if t.T == abi.TupleTy {
		for _, elem := range t.TupleElems {
			if IsDynamicType(*elem) {
				return true
			}
		}
		return false
	}
	return t.T == abi.StringTy || t.T == abi.BytesTy || t.T == abi.SliceTy || (t.T == abi.ArrayTy && IsDynamicType(*t.Elem))
}

// getTypeSize returns the size that this type needs to occupy.
// We distinguish static and dynamic types. Static types are encoded in-place
// and dynamic types are encoded at a separately allocated location after the
// current block.
// So for a static variable, the size returned represents the size that the
// variable actually occupies.
// For a dynamic variable, the returned size is fixed 32 bytes, which is used
// to store the location reference for actual value storage.
func GetTypeSize(t abi.Type) int {
	if t.T == abi.ArrayTy && !IsDynamicType(*t.Elem) {
		// Recursively calculate type size if it is a nested array
		if t.Elem.T == abi.ArrayTy || t.Elem.T == abi.TupleTy {
			return t.Size * GetTypeSize(*t.Elem)
		}
		return t.Size * 32
	} else if t.T == abi.TupleTy && !IsDynamicType(t) {
		total := 0
		for _, elem := range t.TupleElems {
			total += GetTypeSize(*elem)
		}
		return total
	}
	return 32
}

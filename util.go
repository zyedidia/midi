package midi

func convertByteToInt(in [4]byte) int32 {
	return (int32(in[0])<<24 | int32(in[1])<<16 | int32(in[2])<<8 | int32(in[3]))
}

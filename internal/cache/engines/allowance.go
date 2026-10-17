package engines

const (
	allowanceKeyPrefix = "allowance"
)

func allowanceSlot(key string) string {
	return allowanceKeyPrefix + ":{" + key + "}"
}

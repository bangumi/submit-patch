package templates

type StateFilter struct {
	Enum     string
	Readable string
}

var allReadableStateFilter = []StateFilter{
	{
		Enum:     "pending",
		Readable: "待审核",
	},
	{
		Enum:     "reviewed",
		Readable: "已审核",
	},
	{
		Enum:     "all",
		Readable: "全部",
	},
	{
		Enum:     "rejected",
		Readable: "已拒绝",
	},
	{
		Enum:     "accepted",
		Readable: "已接受",
	},
}

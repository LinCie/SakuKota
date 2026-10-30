package subdistrict

const (
	// maxNameLength is the size of the VARCHAR(100) name and slug columns.
	// VARCHAR counts characters, not bytes.
	maxNameLength = 100

	// maxStaticMapImageURLLength is the size of the VARCHAR(255) URL column.
	maxStaticMapImageURLLength = 255

	// maxEstimatedDailySpending is the largest value a DECIMAL(10,2) column holds.
	maxEstimatedDailySpending = 99999999.99
)

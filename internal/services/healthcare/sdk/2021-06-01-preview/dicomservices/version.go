package dicomservices

import "fmt"

const defaultApiVersion = "2021-06-01-preview"

func userAgent() string {
	return fmt.Sprintf("pandora/dicomservices/%s", defaultApiVersion)
}

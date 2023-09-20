package transformer

type constError string

func (err constError) Error() string {
	return string(err)
}

const (
	UnsupportedRecordTypeError = constError("Record does not support")
)

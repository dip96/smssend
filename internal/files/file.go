package files

type FileHandler interface {
	Open(filename string) error
	GetTextSms() string
	GetPhoneNumbers() ([]string, error)
	SetCellValue(cell, value string)
	SaveFile() error
	Close() error
}

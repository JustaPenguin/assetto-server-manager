package properties

type CanValidateValue interface {
	ValidateICalValue() error
}

type CanDecodeValue interface {
	DecodeICalValue(string) error
}

type CanDecodeParams interface {
	DecodeICalParams(Params) error
}

type CanEncodeTag interface {
	EncodeICalTag() (string, error)
}

type CanEncodeValue interface {
	EncodeICalValue() (string, error)
}

type CanEncodeName interface {
	EncodeICalName() (PropertyName, error)
}

type CanEncodeParams interface {
	EncodeICalParams() (Params, error)
}

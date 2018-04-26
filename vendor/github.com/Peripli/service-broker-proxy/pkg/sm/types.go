package sm

//TODO: once SM changes are merged we should import the rest types from SM here and use them to avoid double maintance

// BrokerList broker struct
type BrokerList struct {
	Brokers []Broker `json:"brokers"`
}

// Broker broker struct
type Broker struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	BrokerURL   string       `json:"broker_url"`
	Credentials *Credentials `json:"credentials,omitempty"`
}

// Basic basic credentials
type Basic struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// Credentials credentials
type Credentials struct {
	Basic *Basic `json:"basic,omitempty"`
}

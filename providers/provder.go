package providers

type ServerProvider interface {
	GetServerList() ([]*Server, error)
}

type Server struct {
	Server       string `json:"server"`
	LocalAddress string `json:"local_address"`
	LocalPort    int    `json:"local_port"`
	Timeout      int    `json:"timeout"`
	Workers      int    `json:"workers"`
	ServerPort   int    `json:"server_port"`
	Password     string `json:"password"`
	Method       string `json:"method"`
	Plugin       string `json:"plugin"`
	PingSpeed    float64
}

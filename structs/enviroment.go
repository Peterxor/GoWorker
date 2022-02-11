package structs

type EnviromentModel struct {
	Database         database
	ConcurrentAmount int
	RabbitMQ         rabbitmq
	Log              log
	Email            email
	Server           server
	Router           router
}

type server struct {
	AppAPI string
}

type database struct {
	Client      string
	MaxIdle     uint
	MaxLifeTime string
	MaxOpenConn uint
	User        string
	Password    string
	Host        string
	Db          string
	Params      string
	Port        string
	LogEnable   int
}

type rabbitmq struct {
	Domain string
}

type log struct {
	ElkEnable      int
	ElkIndex       string
	ElkURL         string
	LogstashEnable int
	LogstashURL    string
	LogstashIndex  string
}

type email struct {
	APIUrl string
}

type router struct {
	Port int
}

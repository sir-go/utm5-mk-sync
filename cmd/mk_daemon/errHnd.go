package main

func eh(err error, msg ...string) {
	if err != nil {
		LOG.Error(err, msg)
		panic(err)
	}
}

func ehSkip(err error, msg ...string) {
	if err != nil {
		LOG.Error(err, msg)
	}
}

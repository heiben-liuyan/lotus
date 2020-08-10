package report

var defaultReporter = NewReporter(100)

// set a report url, if noset, it will drop the data by sending.
func SetReportUrl(url string) {
	defaultReporter.SetUrl(url)
}

// unblock and send to the report buffer
func SendReport(data []byte) {
	defaultReporter.Send(data)
}

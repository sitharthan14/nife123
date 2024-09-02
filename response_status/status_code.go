package responsestatus




func Status(code float64)(string){
	var Code int
	Code = int(code)
	arr := [15]string{
		"UNKNOWN",
		"RUNNING",
		"STOPPED",
		"PENDING",
		"WAITING",
		"DELETE",
		"DELETED",
		"FAILED",
		"SCHEDULINGFAILED",
		"IMAGENOTFOUND",
		"EVICTED",
	}
	return arr[Code]
}

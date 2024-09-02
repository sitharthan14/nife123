package duplo

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"github.com/nifetency/nife.io/helper"
)

func GetPort(tenantId, BaseAdrress, appName string) (string,error) {

	    res, err := DuploHTTPGetRequest(BaseAdrress+"/subscriptions/","GetLBConfigurations", tenantId)

	if err != nil {
		return "",err
	}

	body, _ := ioutil.ReadAll(res.Body)

	var result []map[string]interface{}
	err = json.Unmarshal(body, &result)
	if err != nil {
		log.Println(err)
		return "",err
	}

	for _, j := range result {
		port, err := helper.GetPort(j, appName)
		if err != nil {
			log.Println(err)
			return "",err
		}
		return port, nil
	}

	return "", nil
	
}
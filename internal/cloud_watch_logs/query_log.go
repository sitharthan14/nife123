package cloudwatchlogs

import (
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/nifetency/nife.io/api/model"
	database "github.com/nifetency/nife.io/internal/pkg/db/mysql"
)

type QueryLog struct {
	LastSyncTime string `json:"last_sync_time"`
}

func GetLastSyncLogs() (QueryLog, error) {
	var last_sync_time QueryLog

	querystring := "SELECT last_sync_time FROM domain_logs ORDER BY last_sync_time DESC limit 1"
	selDB, err := database.Db.Query(querystring)
	if err != nil {
		return QueryLog{}, err
	}
	defer selDB.Close()
	for selDB.Next() {
		err = selDB.Scan(&last_sync_time.LastSyncTime)
		if err != nil {
			return QueryLog{}, err
		}
	}

	return last_sync_time, err
}

func InsertLog(result QueryResult) (QueryLog, error) {

	statement, err := database.Db.Prepare("Insert into domain_logs(id,message,query_name,resolver_ip,timestamp,response_code,last_sync_time,query_type)values(?,?,?,?,?,?,?,?)")
	if err != nil {
		return QueryLog{}, err
	}

	_, err = statement.Exec(uuid.New(), result.Message, result.QueryName, result.ResolverIp, result.Timestamp, result.ResponseCode, time.Now().UTC(), result.QueryType)
	if err != nil {
		return QueryLog{}, err
	}

	return QueryLog{}, nil

}

func GetQueryLogs(result model.GetQueryLog) []*model.QueryLogOutput {

	var array []*model.QueryLogOutput
	var queryfield model.QueryLogOutput
	query := "SELECT resolver_ip,COUNT(*) count FROM domain_logs where query_name= '" + result.HostName + "' and timestamp between '" + result.StartTime + "' and '" + result.EndTime + "' group by resolver_ip"

	selDB, err := database.Db.Query(query)
	if err != nil {
		return make([]*model.QueryLogOutput, 0)
	}
	defer selDB.Close()
	for selDB.Next() {
		err = selDB.Scan(&queryfield.ResolverIP, &queryfield.Times)
		if err != nil {
			return make([]*model.QueryLogOutput, 0)
		}
		res := model.QueryLogOutput{

			ResolverIP: queryfield.ResolverIP,
			Times:      queryfield.Times,
		}
		array = append(array, &res)
	}

	return array

}

// func InsertClientSideLog(logs model.ClientSideLogs, user_id string) error {

// 	statement, err := database.Db.Prepare("Insert into logs(id,message,level,timestamp,user_id)values(?,?,?,?,?)")
// 	if err != nil {
// 		return err
// 	}

// 	_, err = statement.Exec(uuid.New(), logs.Message, logs.Level, time.Now().UTC(), user_id)
// 	if err != nil {
// 		return err
// 	}

// 	return nil

// }

// func GetClientSideLog(userId string) ([]*model.GetClientSideLogs, error) {

// 	query := `SELECT id, message, level, timestamp, user_id FROM logs where user_id = ?`

// 	selDB, err := database.Db.Query(query, userId)
// 	if err != nil {
// 		return []*model.GetClientSideLogs{}, err
// 	}
// 	defer selDB.Close()

// 	var logs []*model.GetClientSideLogs

// 	for selDB.Next() {
// 		var log model.GetClientSideLogs
// 		err = selDB.Scan(&log.ID, &log.Message, &log.Level, &log.TimeStamp, &log.UserID)
// 		if err != nil {
// 			return []*model.GetClientSideLogs{}, nil
// 		}
// 		logs = append(logs, &log)
// 	}

// 	return logs, nil

// }



func LogFileExists(name string) bool {
    if _, err := os.Stat(name); err != nil {
       if os.IsNotExist(err) {
            return false
        }
    }
    return true
}
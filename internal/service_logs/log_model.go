package servicelogs

import (
	"time"

	"fmt"
	"os"

	"github.com/google/uuid"
	database "github.com/nifetency/nife.io/internal/pkg/db/mysql"

	"github.com/alecthomas/log4go"
	log "github.com/alecthomas/log4go"
)

type MethodLogs struct {
	Email, Module, MethodName, Description, DetailedDescription string
}

var (
	WarningLogger *log.Logger
	InfoLogger    *log.Logger
	ErrorLogger   *log.Logger
)

func Init() {
	_, err := os.Stat("nife-logs")
	if os.IsNotExist(err) {
		err = os.Mkdir("nife-logs", 0755)
		if err != nil {
			fmt.Println(err)
		}
	}

	// log.LoadConfiguration("../../log-config.xml")

	// file, err := os.OpenFile("nife-logs/logs.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	// if err != nil {
	// 	log.Println(err)
	// }

	// InfoLogger = log.Info("INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	// InfoLogger = InfoLogger
	// WarningLogger = log.New(file, "WARNING: ", log.Ldate|log.Ltime|log.Lshortfile)
	// ErrorLogger = log.New(file, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
}

func (log *MethodLogs) CreateLog() error {
	statement, err := database.Db.Prepare("INSERT INTO service_logs(id,email,module_name, method_name, description, detailed_description, createdAt) VALUES(?,?,?,?,?,?,?)")
	if err != nil {
		return err
	}

	id := uuid.NewString()
	_, err = statement.Exec(id, log.Email, log.Module, log.MethodName, log.Description, log.DetailedDescription, time.Now())
	if err != nil {
		return err
	}

	return nil

}

func SuccessLog(module, methodName, message, userId string) {
	log4go.Info("Module: "+module+", MethodName: "+methodName+", Message: "+message+", user: %s", userId)
}
func ErrorLog(module, methodName, userId string, errMessage error) {
	log4go.Error("Module: "+module+", MethodName: "+methodName+", Message: %s user:%s", errMessage.Error(), userId)
}

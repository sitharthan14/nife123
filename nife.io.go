package main

import (
	"context"
	"log"
	"net/http"

	"os"
	"strconv"

	"github.com/alecthomas/log4go"
	"github.com/gorilla/mux"
	"github.com/nifetency/nife.io/api"
	"github.com/nifetency/nife.io/api/generated"
	"github.com/nifetency/nife.io/config"
	"github.com/nifetency/nife.io/docs"
	"github.com/nifetency/nife.io/internal/auth"
	customerdisplayimage "github.com/nifetency/nife.io/internal/customer_display_image"
	domainlogs "github.com/nifetency/nife.io/internal/domain_logs"
	emailverificationcode "github.com/nifetency/nife.io/internal/email_verification_code"
	fileupload "github.com/nifetency/nife.io/internal/fileUpload"
	forgotpassword "github.com/nifetency/nife.io/internal/forgot_password"

	//_ "github.com/nifetency/nife.io/internal/auth"

	session "github.com/nifetency/nife.io/internal/cli_session"
	cloudwatchlogs "github.com/nifetency/nife.io/internal/cloud_watch_logs"
	"github.com/nifetency/nife.io/internal/datadog"
	database "github.com/nifetency/nife.io/internal/pkg/db/mysql"
	planlist "github.com/nifetency/nife.io/internal/plan_list"
	servicelogs "github.com/nifetency/nife.io/internal/service_logs"
	"github.com/nifetency/nife.io/internal/stripes"
	tokenverification "github.com/nifetency/nife.io/internal/token_verification"
	uilogs "github.com/nifetency/nife.io/internal/ui_logs"
	"github.com/nifetency/nife.io/internal/users"
	"github.com/nifetency/nife.io/service"
	"github.com/nifetency/nife.io/utils"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/handler"

	"github.com/go-chi/cors"

	dockerimage "github.com/nifetency/nife.io/internal/dockerImage"

	// "github.com/go-chi/chi"
	"github.com/jasonlvhit/gocron"
	"github.com/joho/godotenv"
	"github.com/nifetency/nife.io/internal/github"
	httpSwagger "github.com/swaggo/http-swagger"
)

func main() {

	err := loadEnv()
	if err != nil {
		log.Fatalf("Error loading .env file")
	}
	scheduler := os.Getenv("SCHEDULER_TIMER_SECS")
	time, _ := strconv.Atoi(scheduler)

	gocron.Every(uint64(time)).Seconds().Do(cloudwatchlogs.Cloudwatchlogs)

	gocron.Start()

	port := utils.GetEnv("PORT", "8080")
	appConf := config.LoadEnvironmentVars()
	database.ConnectDB(appConf)
	//database.Migrate()
	config := generated.Config{Resolvers: &api.Resolver{}}
	handler := registerRoutes(config)

	swaggerInit()
	log.Printf("Connected to http://localhost:%s/ for Nife.io backend", port)
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"}, // All origins
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token", "X-Requested-With", "Access-Control-Allow-Origin"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
	})

	// Where ORIGIN_ALLOWED is like `scheme://dns[:port]`, or `*` (insecure)
	// headersOk := handlers.AllowedHeaders([]string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token","Access-Control-Allow-Origin"})
	// originsOk := handlers.AllowedOrigins([]string{"*"})
	// methodsOk := handlers.AllowedMethods([]string{"GET", "HEAD", "POST", "PUT", "OPTIONS"})

	// start server listen
	// with error handling
	// log.Fatal(http.ListenAndServe(":" +port, handlers.CORS(originsOk, headersOk, methodsOk)(handler)))
	fileHandler := http.StripPrefix("/release", http.FileServer(http.Dir("/release")))
	http.Handle("/release", fileHandler)

	servicelogs.Init()
	log4go.LoadConfiguration("log-config.xml")

	stripeKey, err := service.GetEnvironmentVariables("STRIPE_KEY")
	if err != nil {
		log4go.Error("Module: main, MethodName: GetEnvironmentVariables, Message: %s ", err.Error())
	}
	log4go.Info("Module: main, MethodName: GetEnvironmentVariables, Message: Fetching stripes key from the Database and setting it as Environment Variables")
	os.Setenv("STRIPE_KEY", stripeKey)
	
	log.Fatal(http.ListenAndServe(":"+port, c.Handler(handler)))
}

func registerRoutes(config generated.Config) http.Handler {
	//  router := chi.NewRouter()

	router := mux.NewRouter()
	server := handler.NewDefaultServer(generated.NewExecutableSchema(config))
	server.AroundOperations(func(ctx context.Context, next graphql.OperationHandler) graphql.ResponseHandler {
		graphql.GetOperationContext(ctx).DisableIntrospection = false
		return next(ctx)
	})

	router.Use(auth.Middleware())

	s := http.StripPrefix("/release/", http.FileServer(http.Dir("./release/")))
	router.PathPrefix("/release/").Handler(s)
	//router.Mount("/swagger", httpSwagger.WrapHandler)
	router.HandleFunc("/swagger", httpSwagger.WrapHandler)
	router.Handle("/graphql", server)

	route := router.PathPrefix("/api/v1").Subrouter()
	route.HandleFunc("/login", users.Login).Methods("POST")
	route.HandleFunc("/refreshToken", users.RefreshToken).Methods("POST")
	route.HandleFunc("/register", users.UserRegister).Methods("POST")
	route.HandleFunc("/cli_sessions", session.CLIUserSession).Methods("POST")
	route.HandleFunc("/cli_sessions/{id}", session.GETCLIUserSession).Methods("GET")
	route.HandleFunc("/plan", planlist.GetPlanList).Methods("GET")
	route.HandleFunc("/domainLogs", domainlogs.DomainLog).Methods("POST")
	route.HandleFunc("/forget_password", forgotpassword.ForgotPassword).Methods("POST")
	route.HandleFunc("/verify_token", forgotpassword.VerifyToken).Methods("GET")
	route.HandleFunc("/reset_password", forgotpassword.ResetPassword).Methods("POST")
	route.HandleFunc("/fileUpload/{name}", fileupload.FileUpload).Methods("POST")
	route.HandleFunc("/instanceKey/{name}", fileupload.InstanceKeyUpload).Methods("POST")
	route.HandleFunc("/ssoLogin", users.SSOSignIn).Methods("POST")
	route.HandleFunc("/ssoSignUp", users.SSOSignUp).Methods("POST")
	route.HandleFunc("/stripesPortal", stripes.HandleCustomerPortal).Methods("POST")
	route.HandleFunc("/findDockerImage", dockerimage.FindDockerImage).Methods("POST")
	route.HandleFunc("/sendVerificationCode", emailverificationcode.SendVerificationCode).Methods("POST")
	route.HandleFunc("/verificationCode", emailverificationcode.VerificationCode).Methods("POST")
	route.HandleFunc("/verifyToken", tokenverification.VerifyAccessToken).Methods("POST")
	route.HandleFunc("/freePlan", stripes.EnableFreePlan).Methods("PUT")
	route.HandleFunc("/profileImage", customerdisplayimage.UploadProfileImage).Methods("POST")
	route.HandleFunc("/profileImage", customerdisplayimage.DeleteProfileImage).Methods("DELETE")
	route.HandleFunc("/login/github", github.GithubLoginHandler).Methods("GET")
	route.HandleFunc("/metrics", service.PrintMetrics).Methods("POST")
	route.HandleFunc("/uilogs", uilogs.UILogs).Methods("POST")
	route.HandleFunc("/dataDog", datadog.GetDataDogGraphs).Methods("POST")
	route.HandleFunc("/kubeCongfigFileUpload/{name}", fileupload.KubeConfigFileUpload).Methods("POST")
	route.HandleFunc("/finOpsGcpKey/{name}", fileupload.FinopsGCPKeyUpload).Methods("POST")

	route = router.PathPrefix("/api/v2").Subrouter()
	route.HandleFunc("/register", users.UserRegisterV2).Methods("POST")
	route.HandleFunc("/userRegister", users.UserRegisterOnBoard).Methods("POST")
	return router

}

// 	router.Route("/api/v1", func(router chi.Router) {

// 		router.Post("/login", users.Login)
// 		router.Post("/refreshToken", users.RefreshToken)
// 		router.Post("/register", users.UserRegister)
// 		router.Post("/cli_sessions", session.CLIUserSession)
// 		router.Get("/cli_sessions/{id}", session.GETCLIUserSession)
// 	})
// 	return router

func swaggerInit() {
	docs.SwaggerInfo.Title = "NIFE.IO SWAGGER API"
	docs.SwaggerInfo.Description = "Nife.io is a backend server for nifectl"
	docs.SwaggerInfo.Version = "1.0"
	docs.SwaggerInfo.Host = utils.GetEnv("SWAGGER_HOST", "localhost:8080")
	docs.SwaggerInfo.Schemes = []string{"http", "https"}
}

func loadEnv() error {
	err := godotenv.Load("env")
	return err
}

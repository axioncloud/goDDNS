package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/Microsoft/go-winio"
	"github.com/axioncloud/goDDNS/backend/types"
	"github.com/denisbrodbeck/machineid"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

var OS_UUID = ""

const CONNECTION_FILE = "../frontend/connection"
const CONNECTION_PIPE = "\\\\.\\pipe\\goddns"

func init() {
	if UUID, ok := machineid.ID(); ok != nil {
		fmt.Printf("ERROR: %s\n", ok)
	} else {
		OS_UUID = UUID
	}
}

var DDNS_PROVIDERS map[string]types.Provider = make(map[string]types.Provider)

func updateDDNSProviders() {
	for k := range DDNS_PROVIDERS {
		delete(DDNS_PROVIDERS, k)
	}

	db := openDB()
	defer closeDB(db)

	command := "SELECT ID, NAME, ADDRESS, SELECTED FROM PROVIDERS;"
	result, err := db.Query(command)
	if err != nil {
		log.Fatalln("There was an issue retrieving providers from the DB")
	} else {
		for result.Next() {
			var selectedProvider types.Provider = *new(types.Provider)
			err := result.Scan(&selectedProvider.UUID, &selectedProvider.NAME, &selectedProvider.URL, &selectedProvider.SELECTED)
			if err != nil {
				log.Fatalln("Error getting providers")
			} else {
				DDNS_PROVIDERS[selectedProvider.UUID] = selectedProvider
			}
		}
	}
}

// getProviders responds with the list of all providers as JSON.
func getSelectedProvider(c *gin.Context) {
	db := openDB()
	defer closeDB(db)

	command := "SELECT ID, NAME, ADDRESS, SELECTED FROM PROVIDERS WHERE SELECTED == 1;"
	result, err := db.Query(command)
	if err != nil {
		log.Fatalln("There was an issue retrieving the selected provider in the DB")
	} else {
		for result.Next() {
			var selectedProvider types.Provider = *new(types.Provider)
			err := result.Scan(&selectedProvider.UUID, &selectedProvider.NAME, &selectedProvider.URL, &selectedProvider.SELECTED)
			if err != nil {
				log.Fatalln("Error getting selected provider")
			}
			c.JSON(http.StatusOK, selectedProvider)
		}
	}
}

func getRunRESTServer() bool {
	db := openDB()
	defer closeDB(db)

	command := "SELECT NAME, VALUE FROM CONFIG WHERE NAME == 'RUN_REST_SERVER';"
	result, err := db.Query(command)
	if err != nil {
		log.Fatalln("There was an issue retrieving the selected provider in the DB")
	} else {
		for result.Next() {
			var config types.ConfigItem = *new(types.ConfigItem)
			err := result.Scan(&config.NAME, &config.VALUE)
			if err != nil {
				log.Fatalln("Error getting selected provider")
			} else if config.VALUE == "YES" {
				return true
			} else {
				return false
			}
		}
	}
	return false
}

// getProviders responds with the list of all providers as JSON.
func getProviders(c *gin.Context) {
	query := c.Request.URL.Query()
	query_id := strings.Join(query["id"], "")
	if query_id == "" {
		c.JSON(http.StatusOK, DDNS_PROVIDERS)
	} else {
		c.JSON(http.StatusOK, DDNS_PROVIDERS[query_id])
	}
}

// postProviders adds a new provider to the provider list in the DB.
func postProviders(c *gin.Context) {

	providerStr := c.PostForm("provider")
	urlStr := c.PostForm("url")

	if providerStr == "" || urlStr == "" {
		c.Status(http.StatusBadRequest)
	} else {
		db := openDB()
		defer closeDB(db)

		command := fmt.Sprintf("INSERT INTO PROVIDERS(ID, NAME, ADDRESS) VALUES('%s','%s','%s')", uuid.New().String(), providerStr, urlStr)
		result, err := db.Exec(command)
		if err != nil {
			log.Fatalln("There was an issue inserting the provider to the DB")
		} else {
			numrows, err := result.RowsAffected()
			if err != nil {
				log.Fatalln("Error getting number of rows affected")
			} else {
				log.Printf("Rows affected: %v\n", numrows)
			}
		}
		c.Status(http.StatusOK)
	}
}

// getOSUUID responds with the OS UUID.
func getOSUUID(c *gin.Context) {
	query := c.Request.URL.Query()
	query_id := strings.Join(query["id"], "")
	if query_id == "" {
		c.IndentedJSON(http.StatusOK, struct{ UUID string }{OS_UUID})
	} else {
		c.IndentedJSON(http.StatusOK, struct{ UUID string }{OS_UUID})
	}
}

// getOpenUI opens the frontend.
func getOpenUI(c *gin.Context) {
	query := c.Request.URL.Query()
	query_id := strings.Join(query[""], "")
	if query_id == "" {
		c.AbortWithStatus(http.StatusOK)
	} else {
		c.AbortWithStatus(http.StatusOK)
	}
	openFrontend()
}

// getRestart not implemented.
func getRestart(c *gin.Context) {
	c.Status(http.StatusNotImplemented)
}

// getShutdown not implemented.
func getShutdown(c *gin.Context) {
	c.Status(http.StatusNotImplemented)
}

// getRoot not implemented.
func getRoot(c *gin.Context) {
	c.Status(http.StatusNotImplemented)
}

func openFrontend() {
	os.Chdir("../frontend/")
	wd, _ := os.Getwd()
	log.Println(wd)

	cmd := exec.Command("nwjs-sdk/nw.exe", ".")

	cmd.Env = os.Environ()
	cmd.Dir = wd
	if err := cmd.Start(); err != nil {
		log.Println("Error:", err)
	}
	os.Chdir("../backend/")
}

func openDB() *sql.DB {
	db, err := sql.Open("sqlite", "config.db")
	if err != nil {
		log.Fatalln(err)
	} else {
		db.Ping()
		log.Println("DB opened")
	}
	return db
}

func closeDB(db *sql.DB) {
	err := db.Close()
	if err != nil {
		log.Fatalln(err)
	} else {
		log.Println("DB closed")
	}
}

func main() {
	updateDDNSProviders()

	err := os.Remove(CONNECTION_FILE)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		log.Fatalln(err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	openFrontend()

	router := gin.Default()

	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"PUT", "PATCH", "GET", "DELETE", "POST"},
		AllowHeaders:     []string{"Origin"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		AllowOriginFunc: func(origin string) bool {
			//return origin == "http://localhost"
			return true
		},
		MaxAge: 12 * time.Hour,
	}))

	router.GET("/selectedProvider", getSelectedProvider)
	router.GET("/providers", getProviders)
	router.GET("/osuuid", getOSUUID)
	router.GET("/openui", getOpenUI)
	router.GET("/shutdown", getShutdown)
	router.GET("/restart", getRestart)

	router.GET("/", getRoot)
	router.POST("/", getRoot)
	router.PUT("/", getRoot)
	router.HEAD("/", getRoot)

	router.POST("/providers", postProviders)

	router.StaticFile("/favicon.ico", "../frontend/goddns.ico")

	srvr := &http.Server{
		Addr:    "localhost:65000",
		Handler: router,
	}

	//Listen on Pipe for windows OS
	//Listen on unix socket for *nix/MacOS
	if runtime.GOOS == "windows" {
		listener, err := winio.ListenPipe(CONNECTION_PIPE, nil)

		if err != nil {
			log.Fatalln(err)
		}
		go func() {
			if err := srvr.Serve(listener); err != nil && err != http.ErrServerClosed {
				log.Fatalf("listen: %s\n", err)
				listener.Close()
			}
		}()
	} else {
		listener, err := net.Listen("unix", CONNECTION_FILE)

		if err != nil {
			log.Fatalln(err)
		}
		go func() {
			if err := srvr.Serve(listener); err != nil && err != http.ErrServerClosed {
				log.Fatalf("listen: %s\n", err)
				listener.Close()
			}
		}()
	}

	if getRunRESTServer() {
		listener, err := net.Listen("tcp4", srvr.Addr)

		if err != nil {
			log.Fatalln(err)
		}
		go func() {
			if err := srvr.Serve(listener); err != nil && err != http.ErrServerClosed {
				log.Fatalf("listen: %s\n", err)
				listener.Close()
			}
		}()
	}

	<-ctx.Done()

	// Restore default behavior on the interrupt signal and notify user of shutdown.
	stop()
	log.Println("shutting down gracefully, press Ctrl+C again to force")

	// The context is used to inform the server it has 5 seconds to finish
	// the request it is currently handling
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srvr.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown: ", err)
	}

	log.Println("Server exiting")
}

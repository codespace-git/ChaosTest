package main

import (
    "database/sql"
    "log"
    "time"
    "math/rand"
    "os"
    "os/signal"
    "syscall"
    "context"
    "fmt"
    "net/http"
    "sync"

    c "proxy-manager/constants"
    toxiproxy "github.com/Shopify/toxiproxy/v2/client"
    _ "modernc.org/sqlite"
)

//constants
var baseClientURL = c.BaseClientURL
var dbPath = c.DbPath
var maxtries = c.MaxTries
var timeout_up = c.TimeoutUp
var timeout_down = c.TimeoutDown
var eventPollInterval = c.EventPollInterval
var baseDelay = c.BaseDelay
var healthCheckPort = c.HealthCheckPort
var maxerrorcount = c.MaxErrorCount

// config
var proxyConfig = c.ProxyConfig
var toxics = c.Toxics

type ProxyService struct {
    client *toxiproxy.Client
    db     *sql.DB
    ctx    context.Context
    cancel context.CancelFunc
    closeOnce sync.Once
}

type Event struct {
    ID        int
    EventType string
    OldValue  int
    NewValue  int
    Timestamp string
    Processed int
}

func retryOperation(operation func() error) error {
    var err error
    for i := 0; i < maxtries; i++ {
        if err = operation(); err == nil {
            return nil
        }
        
        if i < maxtries-1 {
            time.Sleep(baseDelay * time.Duration(1<<i))
        }
    }
    return fmt.Errorf("operation failed after %d retries with error: %v", maxtries, err)
}

func CreatedbConnection() (*sql.DB, error) {
    var db *sql.DB

    if err:= retryOperation(func() error{
        var err error
        db,err = sql.Open("sqlite",dbPath)
        return err
    });err!=nil{
        return nil,err
    }

    return db,nil
}

func NewProxyService() (*ProxyService, error) {

    db,err := CreatedbConnection()
    if err!=nil{
        return nil,fmt.Errorf("failed to connect with db: %w", err)
    }

    ctx, cancel := context.WithCancel(context.Background())
    client := toxiproxy.NewClient(baseClientURL)
    ps := &ProxyService{
        client: client,
        db:     db,
        ctx:    ctx,
        cancel: cancel,
    }
    
    if err = ps.initializeDatabase(); err != nil {
        return nil, fmt.Errorf("failed to initialize database: %w", err)
    }
    
    return ps, nil
}

func (ps *ProxyService) retryOperation(operation func() error) error {
    var err error
    for i := 0; i < maxtries; i++ {
        if err = operation(); err == nil {
            return nil
        }
        
        if i < maxtries-1 {
            time.Sleep(baseDelay * (1<<i)) 
        }
    }
    return fmt.Errorf("operation failed after %d retries with error: %v", maxtries, err)
}

func (ps *ProxyService) initializeDatabase() error {
    query := `
        PRAGMA journal_mode=WAL;
        
        CREATE TABLE IF NOT EXISTS control (
            id INTEGER PRIMARY KEY,
            count INTEGER DEFAULT 0,
            data_size INTEGER DEFAULT 1,
            inject INTEGER DEFAULT 0
        );

        INSERT OR IGNORE INTO control (id, count, data_size, inject) 
        VALUES (1, 0, 1, 0);

        CREATE TABLE IF NOT EXISTS events (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            event_type TEXT NOT NULL,
            old_value INTEGER,
            new_value INTEGER,
            timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
            processed INTEGER DEFAULT 0
        );
        
        CREATE TRIGGER IF NOT EXISTS inject_change_trigger
        AFTER UPDATE OF inject ON control
        WHEN NEW.inject != OLD.inject
        BEGIN
            INSERT INTO events (event_type, old_value, new_value)
            VALUES ('inject_changed', OLD.inject, NEW.inject);
        END;	
    `
    
    return ps.retryOperation(func() error {
        _, err := ps.db.Exec(query)
        return err
    })
}



func (ps *ProxyService) createProxies() error {
    for _, cfg := range proxyConfig {
        if proxy,err:= ps.client.Proxy(cfg.Name);err!=nil && proxy==nil{
        err := ps.retryOperation(func() error {
            _, err:= ps.client.CreateProxy(cfg.Name, cfg.Listen, cfg.Upstream)
            return err
        })
        if err != nil {
            return fmt.Errorf("failed to create proxy %s: %w", cfg.Name, err)
        }
    }
    }
    return nil
}



func (ps *ProxyService) getState() (int, int, error) {
    count, size := 0, 1

    err := ps.retryOperation(func() error {
        return ps.db.QueryRow("SELECT count, data_size FROM control WHERE id = 1").
        Scan(&count, &size)
    })

    return count, size, err
}


func (ps *ProxyService) getEvents() ([]Event, error) {
    var events []Event
    
    err := ps.retryOperation(func() error {
       
        tx, err := ps.db.Begin()
        if err != nil {
            return fmt.Errorf("failed to begin transaction: %w", err)
        }
        defer tx.Rollback()


        rows, err := tx.Query(`
            SELECT id, event_type, old_value, new_value, timestamp, processed 
            FROM events 
            ORDER BY timestamp ASC
        `)
        if err != nil {
            return err
        }
        defer rows.Close()
        
        events = nil 
        for rows.Next() {
            var event Event
            err := rows.Scan(&event.ID, &event.EventType, &event.OldValue, 
                           &event.NewValue, &event.Timestamp, &event.Processed)
            if err != nil {
                return err
            }
            events = append(events, event)
        }
        
        if err = rows.Err(); err != nil {
            return err
        }

     
        return tx.Commit()
    })
    
    return events, err
}
func (ps *ProxyService) removeEvents(eventID int) error {
    return ps.retryOperation(func() error {
        _, err := ps.db.Exec("DELETE FROM events WHERE id = ?", eventID)
        return err
    })
}
func (ps *ProxyService) processEvents() error {
    events, err := ps.getEvents()
    if err != nil {
        return fmt.Errorf("failed to fetch events: %w", err)
    }
    if len(events) == 0 {
        return nil 
    }   

    for _, event := range events {
        if event.Processed == 0 {
            log.Printf("Processing event: %v", event)
            switch event.EventType {
            case "inject_changed":
                if event.NewValue == 1 {
                    ps.injectToxics()
            } else {
                ps.removeToxics()
            }
            default:
                log.Printf("Unknown event type: %s", event.EventType)
        }
        err := ps.retryOperation(func() error {
            _, err := ps.db.Exec("UPDATE events SET processed = 1 WHERE id = ?", event.ID)
            return err
        })
        if err != nil {
            log.Printf("Failed to mark event %d as processed: %v", event.ID, err)
        }
    }

        if err := ps.removeEvents(event.ID); err != nil {
            log.Printf("Failed to remove event %d: %v", event.ID, err)
        }
        
    }
    return nil
}

func (ps *ProxyService) removeToxicsForProxy(proxy *toxiproxy.Proxy) {
       
    for _, toxic := range toxics {
        
		err := ps.retryOperation(func() error {

			return proxy.RemoveToxic(toxic)
		})
		if err != nil {
			log.Printf("Failed to remove toxic %s from proxy %s: %v", toxic, proxy.Name, err)
			continue
		}
       
            proxy.Save()
            log.Printf("removed toxic %s from proxy %s", toxic, proxy.Name)
        
    }
}

func (ps *ProxyService) injectToxics() {
    for _, cfg := range proxyConfig {
        var proxy *toxiproxy.Proxy
		err := ps.retryOperation(func() error {
            var err error
            
        proxy, err = ps.client.Proxy(cfg.Name)
        return err
		})
		if err != nil {
			log.Printf("Failed to get proxy %s: %v", cfg.Name, err)
			continue
		}

        
        ps.removeToxicsForProxy(proxy)

        hasToxic := false
        var toxiclist []toxiproxy.Toxic
        err = ps.retryOperation(func() error{
        var err error
        toxiclist, err = proxy.Toxics()
        return err
    })
        if err!=nil{
            log.Printf("error fetching toxics from proxy %s: %v", cfg.Name,err)
        }else{
            hasToxic = len(toxiclist) > 0
        }
       
    
    if !hasToxic {
        var err error
        if rand.Intn(2) == 0 {
            err = ps.retryOperation(func() error {
                _, err := proxy.AddToxic("toxic_timeout_up", "timeout", "upstream", 1.0,
                    toxiproxy.Attributes{"timeout": timeout_up})
                return err
            })
        } else {
            err = ps.retryOperation(func() error {
                _, err := proxy.AddToxic("toxic_timeout_down", "timeout", "downstream", 1.0,
                    toxiproxy.Attributes{"timeout": timeout_down})
                return err
            })
        }
        if err != nil {
            log.Printf("Failed to add toxic to proxy %s: %v", cfg.Name, err)
            continue
        }
        proxy.Save()
    }
}
}


func (ps *ProxyService) removeToxics() {
    for _, cfg := range proxyConfig {
        var proxy *toxiproxy.Proxy
		err := ps.retryOperation(func() error {
        var err error
        proxy, err = ps.client.Proxy(cfg.Name)
        return err
		})
		if err != nil {
			log.Printf("Failed to get proxy %s: %v", cfg.Name, err)
			continue
		}
        
        ps.removeToxicsForProxy(proxy)
        
    }
}

func (ps *ProxyService) deleteProxies() {
	
    for _, cfg := range proxyConfig {
        var proxy *toxiproxy.Proxy
		err := ps.retryOperation(func() error {
			var err error
			proxy, err = ps.client.Proxy(cfg.Name)
			return err
		})
		if err != nil {
			log.Printf("Failed to get proxy %s: %v", cfg.Name, err)
			continue
		}
        proxy.Delete()
        log.Printf("deleted proxy %s",proxy.Name)
    }
}

func (ps *ProxyService) Run() error {
    if err := ps.createProxies(); err != nil {
        return fmt.Errorf("failed to create proxies: %w", err)
    }
    errorcount :=0
    log.Println("Proxy service started successfully")
    
    for {
        select {
        case <-ps.ctx.Done():
            log.Println("Shutting down proxy service...")
            ps.deleteProxies()
            return nil
            
        default:
            if err := ps.processEvents(); err != nil {
                log.Printf("Failed to process events: %v", err)
                errorcount++
                if errorcount >= maxerrorcount{
                    ps.deleteProxies()
                    return fmt.Errorf("error limit reached")
                }
            }

            count, size, err := ps.getState()
            if err != nil {
                return fmt.Errorf("failed to fetch state of db: %w",err)
            }
            if count == size {
                ps.deleteProxies()
                log.Println("Service no longer required, exiting now")
                return nil
            }
        }
        time.Sleep(eventPollInterval)
    }
}

func (ps *ProxyService) Close() {
    ps.cancel()
    ps.db.Close()
}
func (ps *ProxyService) Exit(code int) {
    
    ps.closeOnce.Do(func(){
    ps.Close()
    os.Exit(code)
    })
}

func startHealthServer(ctx context.Context) {
    http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        w.Write([]byte("OK"))
    })
    
    server := &http.Server{Addr: healthCheckPort}
    
    go func() {
        log.Printf("HealthCheckServer running on port %s",healthCheckPort)
        if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            log.Printf("Health check server error: %v", err)
        }
    }()

    go func() {
        <-ctx.Done()
        log.Println("Shutting down health check server...")
        shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
        defer cancel()
        server.Shutdown(shutdownCtx)
    }()
}

func main() {
    ps, err := NewProxyService()
    if err != nil {
        log.Fatalf("Failed to create proxy service: %v", err)
    }
    
   startHealthServer(ps.ctx)

    c := make(chan os.Signal, 1)
    signal.Notify(c, os.Interrupt, syscall.SIGTERM)
    
    go func() {
        <-c
        log.Println("Received shutdown signal")
        ps.Exit(0)
    }()
    

    if err := ps.Run(); err != nil {
        log.Printf("Proxy service failed: %v", err)
        ps.Exit(1)
    }
    ps.Exit(0)
}
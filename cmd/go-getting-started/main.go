//func main() {
//	port := os.Getenv("PORT")
//
//	if port == "" {
//		log.Fatal("$PORT must be set")
//	}
//
//	router := gin.New()
//	router.Use(gin.Logger())
//	router.LoadHTMLGlob("templates/*.tmpl.html")
//	router.Static("/static", "static")
//
//	router.GET("/", func(c *gin.Context) {
//		c.HTML(http.StatusOK, "index.tmpl.html", nil)
//	})
//
//	router.Run(":" + port)
//}
package main

import (
	"database/sql"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"time"

	_ "github.com/lib/pq"
)

var db *sql.DB = nil

const (
	width, height = 600, 320            // canvas size in pixels
	cells         = 100                 // number of grid cells
	xyrange       = 30.0                // axis ranges
	xyscale       = width / 2 / xyrange // pixels per x or y unit
	zscale        = height * 0.4        // pixels per z unit
	angle         = math.Pi / 6         // angle of x, y axes =30
)

var sin30, cos30 = math.Sin(angle), math.Cos(angle) // sin(30),cos(30)

func main() {
	var errd error
	port := os.Getenv("PORT")

	if port == "" {
		log.Fatal("$PORT must be set")
	}

	db, errd = sql.Open("postgres", os.Getenv("DATABASE_URL"))
	if errd != nil {
		log.Fatalf("Error opening database: %q", errd)
	}
	http.HandleFunc("/", handler)  //each request calls handler
	http.HandleFunc("/db", dbFunc) //each request calls handler
	log.Fatal(http.ListenAndServe("0.0.0.0:"+port, nil))
}

func handler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "image/svg+xml")

	//fmt.Fprintf(w, "%s %s %s\n", r.Method, r.URL, r.Proto)
	fmt.Fprintf(w, "<svg xmlns='http://www.w3.org/2000/svg' "+
		"style='stroke: grey; fill: white; stroke-width: 0.7' "+
		"width='%d' height='%d'>", width, height)
	for i := 0; i < cells; i++ {
		for j := 0; j < cells; j++ {
			ax, ay := corner(i+1, j)
			bx, by := corner(i, j)
			cx, cy := corner(i, j+1)
			dx, dy := corner(i+1, j+1)
			if !anyNaNs(ax, ay, bx, by, cx, cy, dx, dy) {
				fmt.Fprintf(w, "<polygon points='%g,%g %g,%g %g,%g %g,%g'/>\n",
					ax, ay, bx, by, cx, cy, dx, dy)
			} else {
				fmt.Errorf("WHY YOU NAN??='%g,%g %g,%g %g,%g %g,%g'",
					ax, ay, bx, by, cx, cy, dx, dy)
			}
		}
	}
	fmt.Fprintln(w, "</svg>")
}

func anyNaNs(ax, ay, bx, by, cx, cy, dx, dy float64) bool {
	return math.IsNaN(ax) || math.IsNaN(ay) || math.IsNaN(bx) || math.IsNaN(by) || math.IsNaN(cx) || math.IsNaN(cy) || math.IsNaN(dx) || math.IsNaN(dy)
}

func corner(i, j int) (float64, float64) {
	// find pint (x,y) at corner of cell (i,j).
	x := xyrange * (float64(i)/cells - 0.5)
	y := xyrange * (float64(j)/cells - 0.5)

	// compure surface height z
	z := f(x, y)

	// protext (x,y,z) isometrically onto the 2-D SVG canvas sx,sy
	sx := width/2 + (x-y)*cos30*xyscale
	sy := height/2 + (x+y)*sin30*xyscale - z*zscale
	return sx, sy
}

func f(x, y float64) float64 {
	r := math.Hypot(x, y) // distance from (0,0)
	return math.Sin(r) / r
}

func dbFunc(w http.ResponseWriter, r *http.Request) {
	if _, err := db.Exec("CREATE TABLE IF NOT EXISTS ticks (tick timestamp)"); err != nil {
		fmt.Fprintln(w, "Error creating database table: %q", err)
		return
	}

	if _, err := db.Exec("INSERT INTO ticks VALUES (now())"); err != nil {
		fmt.Fprintln(w, "Error incrementing tick: %q", err)
		return
	}

	rows, err := db.Query("SELECT tick FROM ticks")
	if err != nil {
		fmt.Fprintln(w, "Error reading tick: %q", err)
		return
	}

	defer rows.Close()
	for rows.Next() {
		var tick time.Time
		if err := rows.Scan(&tick); err != nil {
			fmt.Fprintln(w, "Error scanning tick: %q", err)
			return
		}
		fmt.Fprintln(w, "Read from DB: %s\n", tick.String())
	}
}

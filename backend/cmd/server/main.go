package main
import (
	"fmt"
	"log"
	"net/http"
)
func main(){
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request){
		fmt.Fprintln(w, "Добро пожаловать в Fluxora.")
	})
	http.HandleFunc("/about", func(w http.ResponseWriter, r *http.Request){
		fmt.Fprintln(w, "Fluxora - твой личный финансовый помощник.")
	})
	fmt.Println("Fluxora сервер запущен по http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
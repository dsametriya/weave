package weavedns

import (
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
)

// Parse a URL of the form PUT /name/<identifier>/<ip-address>
func parseUrl(url string) (identifier string, ipaddr string, err error) {
	parts := strings.Split(url, "/")
	if len(parts) != 4 {
		return "", "", errors.New("Unable to parse url: " + url)
	}
	return parts[2], parts[3], nil
}

func ListenHttp(db Zone, port int) {
	http.HandleFunc("/name/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "PUT":
			identifier, weave_ipstr, err := parseUrl(r.URL.Path)
			name := r.FormValue("fqdn")
			prefix := r.FormValue("routing_prefix")
			local_ip := r.FormValue("local_ip")
			if identifier == "" || weave_ipstr == "" || name == "" || prefix == "" || local_ip == "" {
				log.Printf("Invalid request: %s, %s", r.URL, r.Form)
				http.Error(w, "Invalid Request", http.StatusBadRequest)
				return
			}
			ip := net.ParseIP(local_ip)
			if ip == nil {
				log.Printf("Invalid IP in request: %s", local_ip)
				http.Error(w, "Invalid IP in request", http.StatusBadRequest)
				return
			}
			weave_cidr := weave_ipstr + "/" + prefix
			weave_ip, subnet, err := net.ParseCIDR(weave_cidr)
			if err != nil {
				log.Printf("Invalid CIDR in request: %s", weave_cidr)
				http.Error(w, fmt.Sprintf("Invalid CIDR: %s", weave_cidr), http.StatusBadRequest)
				return
			}
			log.Printf("Adding %s (%s) -> %s", name, local_ip, weave_cidr)
			err = db.AddRecord(identifier, name, ip, weave_ip, subnet)
			if err != nil {
				http.Error(w, err.Error(), http.StatusConflict)
				return
			}
		case "DELETE":
			identifier, weave_ipstr, err := parseUrl(r.URL.Path)
			if identifier == "" || weave_ipstr == "" {
				log.Printf("Invalid request: %s, %s", r.URL, r.Form)
				http.Error(w, "Invalid Request", http.StatusBadRequest)
				return
			}
			weave_ip := net.ParseIP(weave_ipstr)
			if weave_ip == nil {
				log.Printf("Invalid IP in request: %s", weave_ipstr)
				http.Error(w, "Invalid IP in request", http.StatusBadRequest)
				return
			}
			log.Printf("Deleting %s (%s)", identifier, weave_ipstr)
			err = db.DeleteRecord(identifier, weave_ip)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
		default:
			log.Println("Unexpected http method", r.Method)
			http.Error(w, "Unexpected http method: "+r.Method, http.StatusBadRequest)
		}
	})
	address := fmt.Sprintf(":%d", port)
	err := http.ListenAndServe(address, nil)
	if err != nil {
		log.Fatal("Unable to create http listener: ", err)
	}
}

package main
import (
    "fmt" 
    "github.com/tidwall/gjson"
)
func main() {
    json := `{"root":{"users":{"user":[{"name":"王五"},{"name":"赵六"}]}}}`
    fmt.Println("root.users.user.name:", gjson.Get(json, "root.users.user.name"))
    fmt.Println("root.users.user.0.name:", gjson.Get(json, "root.users.user.0.name")) 
}

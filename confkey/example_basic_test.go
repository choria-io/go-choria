package confkey_test

import (
	"fmt"
	"os"
	"strings"

	confkey "github.com/choria-io/go-config/confkey"
)

type Config struct {
	Loglevel string   `confkey:"loglevel" default:"warn" validate:"enum=debug,info,warn,error"`
	Mode     string   `confkey:"mode" default:"server" validate:"enum=server,client"`
	Servers  []string `confkey:"servers" type:"comma_split" environment:"SERVERS"`
	Path     []string `confkey:"path" type:"path_split" default:"/bin:/usr/bin"` // can also be colon_split to always split on :
}

func Example_basic() {
	c := &Config{}

	err := confkey.SetStructDefaults(c)
	if err != nil {
		panic(err)
	}

	fmt.Println("Defaults:")
	fmt.Printf("  loglevel: %s\n", c.Loglevel)
	fmt.Printf("  mode: %s\n", c.Mode)
	fmt.Printf("  path: %s\n", strings.Join(c.Path, ","))
	fmt.Println("")

	// here you would read your config file, but lets just fake it
	// and set specific values

	// every call to SetStructFieldWithKey validates what gets set
	err = confkey.SetStructFieldWithKey(c, "loglevel", "error")
	if err != nil {
		panic(err)
	}

	err = confkey.SetStructFieldWithKey(c, "mode", "client")
	if err != nil {
		panic(err)
	}

	// even though we are setting it, if the ENV is set it overrides
	os.Setenv("SERVERS", "s1:1024, s2:1024")
	err = confkey.SetStructFieldWithKey(c, "servers", "s:1024")
	if err != nil {
		panic(err)
	}

	fmt.Println("Loaded:")
	fmt.Printf("  loglevel: %s\n", c.Loglevel)
	fmt.Printf("  mode: %s\n", c.Mode)
	fmt.Printf("  servers: %s\n", strings.Join(c.Servers, ","))
	fmt.Println("")

	// getting a string by name
	fmt.Println("Retrieved:")
	fmt.Printf("  loglevel: %s\n", confkey.StringFieldWithKey(c, "loglevel"))
	fmt.Printf("  servers: %s\n", strings.Join(confkey.StringListWithKey(c, "servers"), ","))
	fmt.Println("")

	// but you can also validate the entire struct if you like, perhaps you
	// set some stuff directly to its fields
	err = confkey.Validate(c)
	if err != nil {
		fmt.Printf("invalid: %s\n", err)
		panic(err)
	}

	fmt.Println("valid")

	// setting a specific bad value yields an error
	err = confkey.SetStructFieldWithKey(c, "loglevel", "fail")
	if err != nil {
		fmt.Printf("invalid: %s\n", err)
	}

	// Output:
	// Defaults:
	//   loglevel: warn
	//   mode: server
	//   path: /bin,/usr/bin
	//
	// Loaded:
	//   loglevel: error
	//   mode: client
	//   servers: s1:1024,s2:1024
	//
	// Retrieved:
	//   loglevel: error
	//   servers: s1:1024,s2:1024
	//
	// valid
	// invalid: Loglevel enum validation failed: 'fail' is not in the allowed list: debug, info, warn, error
}

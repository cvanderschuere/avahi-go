package avahi

import "testing"
import "fmt"
import "time"

func TestBrowseImmediate(t *testing.T){
	results := BrowseServiceImmediate("_musicbox._tcp")
	for _,service := range results{	
		fmt.Println("Name: "+service.Name)
		fmt.Println("Host: "+service.Hostname)
		fmt.Println("Address: "+service.Address)
		fmt.Println("Port:", service.Port)
		fmt.Println("/////////////////////////") //Barrier
	}
}

func TestBrowse(t *testing.T){
	quitChan := make(chan interface{})
	
	resultChan := BrowseService("_musicbox._tcp",quitChan)
	
	//Kill in 10 seconds
	
	go func(){
		<-time.After(5*time.Second)
		fmt.Println("Quit")
	}()
	
	for result := range resultChan{
		fmt.Println("/////////////////////////") //Barrier
		//Iterate over response
		for key,value := range result{
			fmt.Println("Key:",key,"Value:",value);
		}
		
		fmt.Println("/////////////////////////") //Barrier	
	}
	
}
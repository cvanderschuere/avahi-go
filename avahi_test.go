package avahi
//Author: Chris Vanderschuere

import "testing"
import "fmt"
import "time"
import "log"


func TestPublish(t *testing.T){
	quitChan,err := PublishService("Test","_example._tcp",8080,"TEST","Another Test");
	if err != nil{
		log.Fatal(err)
	}
	
	<-time.After(3*time.Second)
	quitChan<-true //Kill process
} 

func TestBrowseImmediate(t *testing.T){
	results := BrowseServiceImmediate("_example._tcp")
	for _,service := range results{	
		fmt.Println("Name: "+service.Name)
		fmt.Println("Host: "+service.Hostname)
		fmt.Println("Address: "+service.Address)
		fmt.Println("Port:", service.Port)
		fmt.Println("TXT:", service.TXT)
		fmt.Println("/////////////////////////") //Barrier
	}
}

func TestBrowse(t *testing.T){
	quitChan := make(chan interface{})
	
	resultChan := BrowseService("_example._tcp",quitChan)
	
	//Kill in 5 seconds
	go func(){
		<-time.After(5*time.Second)
		quitChan<-true
	}()
	
	//Loop through all updates
	for result := range resultChan{
		fmt.Println("/////////////////////////") //Barrier
		//Iterate over response
		for key,value := range result{
			fmt.Println("Key:",key,"Value:",value);
		}
		
		fmt.Println("/////////////////////////") //Barrier	
	}
	
}
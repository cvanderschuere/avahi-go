/*
	A simple wrapper to avahi-publish and avahi-browse
*/
package avahi
//Author: Chris Vanderschuere

import(
	"os/exec"
	"strings"
	"strconv"
	"bufio"
	"io"
	"log"
)

const(
	AVAHI_SERVICE_RESOLVE 	= "="
	AVAHI_SERVICE_DELETE 	= "+"
	AVAHI_SERVICE_ADD 		= "-"
)

type Service struct{
	Name		string
	Hostname 	string
	ServiceType string
	Address 	string
	Port 		int
	TXT			string
}

//
//Publish
//

//Publishes a service with the given paramaters
func PublishService(name,serviceType string, port int, txt ...string)(chan<- interface{}, error){
	//Exec avahi-publish with given paramaters
	args := make([]string,3)
	args[0] = name
	args[1] = serviceType
	args[2] = strconv.Itoa(port)
	args = append(args,txt...)//append on any txt data
	
	cmd := exec.Command("avahi-publish-service",args...)
	killChan := make(chan interface{})
	
	//Start Command
	err := cmd.Start()
	
	if err != nil{
		log.Fatal(err)
	}
	
	//Kill process if command given
	go func(){
		<-killChan
		cmd.Process.Kill()
	}()
	
	
	return killChan,err
}

//
//Browse
//

//Browses the network for a sufficent amount of time and returns services (Blocking)
func BrowseServiceImmediate(serviceType string)(map[string]Service){
	//Exec avahi-browse with given paramaters: use option -t to end when list of services complete
	cmd := exec.Command("avahi-browse","-t","-r",serviceType)
	
	//Read output
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}
	
	//Buffered Reader
	rd := bufio.NewReader(stdout)
	
	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}
	
	services := make(map[string]Service)
	//line,err := rd.ReadString('\n')
	
	//Read strings as they come in until io.EOF
	for {
		line,err := rd.ReadString('\n')
		 if err == io.EOF{
			 break
		 }
		comps := strings.Fields(line)
		 
				
		//Determine type of update
		switch comps[0]{
		case AVAHI_SERVICE_DELETE:
			continue //Alse skip deletions for now
		case AVAHI_SERVICE_RESOLVE: //Resolved Services
			//Add service to array for this item
			newService := parseResolvedService(line,rd)

			services[newService.Name] = newService
			
		case AVAHI_SERVICE_ADD:
			continue //Skip over additions...wait until detailed information
		}	
	}	
	
	//Wait until command finished
	cmd.Wait()
	
	return services
}

//Performs same function as BrowseServiceImmediate(), but monitors services and passes updated information until told to quit (Async)
func BrowseService(serviceType string, quitChan <-chan interface{})(<-chan map[string]Service){
	//Create update chan
	updateChan := make(chan map[string]Service,1)
	
	//Asyncronously monitor for services
	go func(){
		//Send initial snapshot
		services := BrowseServiceImmediate(serviceType)
		updateChan <- services
		
		//Launch constant monitoring
		cmd := exec.Command("avahi-browse","-r",serviceType)
		
		//Read output
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			log.Fatal(err)
		}
	
		//Buffered Reader
		rd := bufio.NewReader(stdout)
		
		if err := cmd.Start(); err != nil {
			log.Fatal(err)
		}
		
		readChan := make(chan string,1)
		//Launch goroutine to read concurrently...send results on chan
		go func(){
			//Read from stdout pipe
			for{
				line,err := rd.ReadString('\n')
				 if err == io.EOF{
					 close(readChan)
					 return
				 }	
				 readChan<-line
			 }
		}()
		
		endChan := make(chan interface{})
		go func(){
			endChan <-cmd.Wait()
		}()
		
		for{
			select {
			case <-quitChan:
				close(updateChan)
				//Kill process and return
				if err := cmd.Process.Kill(); err != nil {
					log.Fatal("failed to kill: ", err)
				}
				return;
			case <-endChan:
				//Command finished
				close(updateChan)
				return
			case line,ok := <- readChan:	
				if ok == false{
					return
				}
			
				comps := strings.Fields(line)
	 
				//Determine type of update
				switch comps[0]{
				case AVAHI_SERVICE_DELETE:
					//Delete service
					splitLine := strings.SplitN(line,"               ",2)
					splitComps := strings.Fields(splitLine[0])
					splitComps = splitComps[3:]
					name := strings.Join(splitComps," ")
					delete(services,name)
					
					updateChan <- services //Send updated services
										
				case AVAHI_SERVICE_RESOLVE: //Resolved Services
					//Add service to array for this item
					newService := parseResolvedService(line,rd)
		
					services[newService.Name] = newService
					
					updateChan <- services //Send update
		
				case AVAHI_SERVICE_ADD:
					//Skip over additions...wait until resolved information
				}
			}
		}
	}()
	
	
	return updateChan
}

//Takes in the first line of a read where '=' is the first character...reads and returns Service
func parseResolvedService(line string, rd *bufio.Reader)(Service){
	var newService Service
	//Split name line
	splitLine := strings.SplitN(line,"               ",2)
	splitComps := strings.Fields(splitLine[0])
	splitComps = splitComps[3:]
	name := strings.Join(splitComps," ")
	
	newService.Name = name
		
	//Hostname: Skip to next line
	line, _ = rd.ReadString('\n')
	comps := strings.Fields(line)
	
	trimmed := strings.Trim(comps[2],"[]")
	newService.Hostname = trimmed
	
	//Address: Skip to next line
	line, _ = rd.ReadString('\n')
	comps = strings.Fields(line)
	
	trimmed = strings.Trim(comps[2],"[]")
	newService.Address = trimmed
		
	//Port: Skip to next line
	line, _ = rd.ReadString('\n')
	comps = strings.Fields(line)

	trimmed = strings.Trim(comps[2],"[]")
	newService.Port, _ = strconv.Atoi(trimmed)
	
	//TXT:
	line, _ = rd.ReadString('\n')
	trimmed = strings.TrimLeft(line,"txt= [")
	trimmed = strings.TrimRight(trimmed,"\n] ")
	
	newService.TXT = trimmed
	return newService
}

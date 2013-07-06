package avahi

import(
	"os/exec"
	"strings"
	"strconv"
	"bufio"
	"io"
	"log"
)

const(
	AVAHI_SERVICE int = iota
	AVAHI_DELETE_SERVICE
	AVAHI_ADD_SERVICE
)

type Service struct{
	Name		string
	Hostname 	string
	ServiceType string
	Address 	string
	Port 		int
	TXT			string
}

type serviceUpdate struct{
	updateType int
	service Service
}

//
//Publish
//

func PublishService(name,serviceType string, port int)(chan<- interface{}, error){
	//Exec avahi-publish with given paramaters
	cmd := exec.Command("avahi-publish-service",name,serviceType,strconv.Itoa(port))
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

//Example Text
/*
+   eth0 IPv4 BeagleBoneMusicBox                            _musicbox._tcp       local
=   eth0 IPv4 BeagleBoneMusicBox                            _musicbox._tcp       local
   hostname = [beaglebone.local]
   address = [192.168.0.199]
   port = [8070]
   txt = ["LivingRoom"]
-   eth0 IPv4 BeagleBoneMusicBox                            _musicbox._tcp       local
*/
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
		case "-":
			continue //Alse skip deletions for now
		case "=": //Resolved Services
			//Add service to array for this item
			var newService Service
			//Split name line
			splitLine := strings.SplitN(line,"               ",2)
			splitComps := strings.Fields(splitLine[0])
			splitComps = splitComps[3:]
			name := strings.Join(splitComps," ")
			
			newService.Name = name
				
			//Hostname: Skip to next line
			line, _ = rd.ReadString('\n')
			comps = strings.Fields(line)
			
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
			
			newService.TXT = "" //Ignore this field
			
			services[newService.Name] = newService
			
		case "+":
			continue //Skip over additions...wait until detailed information
		}	
	}	
	
	//Wait until command finished
	cmd.Wait()
	
	return services
}

//Performs same function as BrowseServiceImmediate(), but monitors services and passes updated information until told to quit
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
				case "-":
					//Delete service
					splitLine := strings.SplitN(line,"               ",2)
					splitComps := strings.Fields(splitLine[0])
					splitComps = splitComps[3:]
					name := strings.Join(splitComps," ")
					delete(services,name)
					
					updateChan <- services //Send updated services
										
				case "=": //Resolved Services
					//Add service to array for this item
					var newService Service
					//Split name line
					splitLine := strings.SplitN(line,"               ",2)
					splitComps := strings.Fields(splitLine[0])
					splitComps = splitComps[3:]
					name := strings.Join(splitComps," ")
		
					newService.Name = name
			
					//Hostname: Skip to next line
					line, _ = rd.ReadString('\n')
					comps = strings.Fields(line)
		
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
		
					newService.TXT = "" //Ignore this field
		
					services[newService.Name] = newService
					
					updateChan <- services //Send update
		
				case "+":
					//Skip over additions...wait until resolved information
				}
			}
		}
	}()
	
	
	return updateChan
}
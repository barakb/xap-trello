## How to install and use with Docker

1. Get the docker  `docker pull barakb/trello`
2. Create configuration file named trello.ini with the following properties
    
    ```bash
    appKey=
    appToken=
    ```
    Values should be taken from https://trello.com/app-key
3. Run the command `docker run  -v ${config_dir}:/trello-conf -p 127.0.0.1:8080:8080  -it barakb/trello:0.1`
   where `${config_dir}` is the directory that contains the trello.ini file
4. In the container run `./serve.sh`
5. Open browser to localhost:8080 you will see the jason there
    
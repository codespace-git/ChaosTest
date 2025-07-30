class Default :
    def __init__(self):

        self._proxy = {
            "search": "6000",
            "weather": "6001", 
            "movie": "6002",
            "calendar": "6003",
            "translator": "6006",
            "calculator": "6004",
            "message": "6005"
        }

        self._services = {
            "search": "search_tool:5000",
            "weather": "weather_tool:5001",
            "movie": "movie_tool:5002", 
            "calendar": "calendar_tool:5003",
            "calculator": "calculator_tool:5004",
            "message": "message_tool:5005",
            "translator": "translator_tool:5006",
            "proxy_mgr": "proxy_mgr:8000"
        }

        self._intervals = {
            "proxy_check_interval": 3,
            "proxy_wait": 30,
            "proxy_mgr_wait": 40,
            "proxy_timeout": 5,
            "fallback_timeout": 10
        }

        self._files = {"prompts":"prompts.json"}

        self.__function_map = {
            "proxy":self.proxies,
            "services":self.services,
            "intervals":self.intervals,
            "files":self.files
        }


    @property
    def services(self):
        return self._services

    @property
    def proxies(self):
        return self._proxy

    @property
    def intervals(self):
        return self._intervals
    
    @property
    def files(self):
        return self._files

    @property
    def all_defaults(self):
        return (self.proxies, self.services, self.intervals, self.files)
    
    def default_items(self,items:list):
        res = []
        for item in items:
          res.append(self.__function_map[item])
        return res
    




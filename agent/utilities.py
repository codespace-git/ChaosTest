import requests
import re
import time


class ToolLimitReachedException(Exception):
    def __init__(self, status_code,start_time,end_time,message):
        self.status_code = status_code
        self.start_time = start_time
        self.end_time = end_time
        self.message = message
        super().__init__(message)


class FileLoadingException(Exception):
    def __init__(self,message):
        self.message = message
        super().__init__(message)


class Utils:
    def __init__(self, proxy_map, service_map,timeout,logger):

        self.proxy_map = proxy_map
        self.service_map = service_map
        self.timeout = timeout
        self.logger = logger
    
    def _get_proxy_status(self,tool_name):

        try:
            response = requests.get(f"http://toxiproxy:{self.proxy_map[tool_name]}/", timeout=self.timeout)
            return response.status_code == 200
        except (requests.exceptions.Timeout,requests.exceptions.ConnectionError) as e:
            self.logger.error(f"Tool {tool_name} not responding in time")
            return False
        except Exception as e:
            self.logger.error(f"Error checking status of {tool_name}: {str(e)}")
            return False

    def _proxy_mgr_status(self,param = None):

        try:
            response = requests.get(f"http://{self.service_map['proxy_mgr']}/health",timeout =self.timeout)
            return response.status_code==200
        except (requests.exceptions.Timeout, requests.exceptions.ConnectionError) as e:
            self.logger.error(f"proxy mgr not responding")
            return False
        except Exception as e:
            self.logger.error(f"Error checking status of proxy mgr: {str(e)}")
            return False
    
    def _classifier(self,status:int):
        if 400<=status<500 :
            return "agent"
        elif 500<=status<600 :
            return "server"
        else :
            return "none"

    def active(self,wait,interval,tool_name = None):
        
        status = self._get_proxy_status if tool_name else self._proxy_mgr_status
        
        if status(tool_name): 
            return True

        wait_time = 0
        self.logger.error(f"{tool_name} Proxy is down,waiting for maximum of {wait}s to restart")
    
        while wait_time < wait:
            if wait_time + interval <= wait:
                sleep_interval = interval
                wait_time += interval
            else:
                sleep_interval = wait - wait_time
                wait_time = wait
        
            time.sleep(sleep_interval)
        
            if status(tool_name):  
                self.logger.info(f"{tool_name} Proxy is up after {wait_time}s,continuing with proxy")
                return True
    
        self.logger.info(f"{tool_name} Proxy remains unreachable")
        return False


    def error_classifier(self,status):
        if isinstance(status,int):
            return self._classifier(status)
        else:
            match = re.search(r'["\']?status["\']\s*:\s*(\d+)', str(status))
            if match:
                status_code= int(match.group(1))
                error_type = self._classifier(status_code)
                return error_type,status_code
            else:
                return "could not resolve error type",599
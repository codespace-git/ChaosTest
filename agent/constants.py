import os

def get_env_var(key, default, cast_fn):
    try:
        return cast_fn(os.getenv(key, str(default)))
    except ValueError:
        return default

InjectPrev = 0
InjectNext = 0
FailCount = 0
SpecialStrings = ["Tool limit reached"]

TOXICPROB = get_env_var("TOXIC_PROB", 0.1, float)
TOOLLIMIT = get_env_var("TOOL_LIMIT", 1, int)
PROMPTLIMIT = get_env_var("PROMPT_LIMIT", 1, int)
ERRORPROB = get_env_var("ERROR_PROB",0.1,float)
import time

def handler(params, context):
    time.sleep(1.4)
    return int(params["input"])

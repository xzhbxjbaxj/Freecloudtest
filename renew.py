import os
import json
import requests
from urllib.parse import urlencode

LOGIN_URL = "https://freecloud.ltd/login"
CONSOLE_URL = "https://freecloud.ltd/member/index"
RENEW_URL_TEMPLATE = "https://freecloud.ltd/server/detail/{}/renew"

HEADERS = {
    "User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36",
    "Referer": "https://freecloud.ltd/login",
    "Origin": "https://freecloud.ltd",
    "Content-Type": "application/x-www-form-urlencoded",
}


def login(session: requests.Session, username: str, password: str) -> bool:
    print(f"🔑 Logging in as {username}...")

    form_data = {
        "username": username,
        "password": password,
        "mobile": "",
        "captcha": "",
        "verify_code": "",
        "agree": "1",
        "login_type": "PASS",
        "submit": "1"
    }

    resp = session.post(LOGIN_URL, data=form_data, headers=HEADERS)
    if resp.status_code != 200:
        print("❌ Login request failed")
        return False

    if "退出登录" not in resp.text and "member/index" not in resp.text:
        print("❌ 登录失败，请检查用户名或密码")
        return False

    print("✅ 登录成功")
    return True


def renew_machine(session: requests.Session, machine_id: int):
    print(f"🔁 尝试为服务器 {machine_id} 续费...")

    form_data = {
        "month": "1",
        "submit": "1",
        "coupon_id": "0",
    }

    renew_url = RENEW_URL_TEMPLATE.format(machine_id)
    resp = session.post(renew_url, data=form_data, headers=HEADERS)

    try:
        result = resp.json()
        msg = result.get("msg", "")
        if msg == "续费成功":
            print(f"✅ 续费成功: {machine_id}")
        elif msg == "请在到期前3天后再续费":
            print(f"⚠️ 提示: {msg}")
        else:
            print(f"❗ 续费异常: {msg}")
    except Exception:
        print("⚠️ 非 JSON 响应:")
        print(resp.text)


def main():
    profiles_env = os.getenv("FC_PROFILES")
    if not profiles_env:
        print("❌ 请设置环境变量 FC_PROFILES")
        return

    try:
        profiles = json.loads(profiles_env)
        if not isinstance(profiles, list):
            profiles = [profiles]
    except Exception as e:
        print(f"❌ JSON 解析错误: {e}")
        return

    for profile in profiles:
        username = profile.get("username")
        password = profile.get("password")
        machines = profile.get("machines", [])

        with requests.Session() as session:
            if login(session, username, password):
                for machine_id in machines:
                    renew_machine(session, machine_id)


if __name__ == "__main__":
    main()

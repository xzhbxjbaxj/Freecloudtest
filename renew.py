import os
import sys
import json
import time
import cloudscraper
from urllib.parse import urlencode

LOGIN_URL = "https://freecloud.ltd/login"
CONSOLE_URL = "https://freecloud.ltd/member/index"
RENEW_URL = "https://freecloud.ltd/server/detail/{}/renew"

HEADERS = {
    "User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36",
    "Referer": "https://freecloud.ltd/login",
    "Origin": "https://freecloud.ltd",
}

def login(scraper, username, password):
    print(f"🔑 Logging in as {username}...")

    payload = {
        "username": username,
        "password": password,
        "mobile": "",
        "captcha": "",
        "verify_code": "",
        "agree": "1",
        "login_type": "PASS",
        "submit": "1"
    }

    try:
        response = scraper.post(LOGIN_URL, data=payload, headers=HEADERS)
        if response.status_code != 200:
            print(f"❌ Login failed with status: {response.status_code}")
            return False

        # 检查是否登录成功
        if "退出登录" not in response.text and "member/index" not in response.text:
            print("❌ Login failed. Please check username/password.")
            return False

        # 访问控制台页面验证会话
        check = scraper.get(CONSOLE_URL)
        if check.status_code != 200:
            print("❌ Session check failed.")
            return False

        print("✅ Login successful.")
        return True
    except Exception as e:
        print(f"❌ Login request failed: {e}")
        return False

def renew_machine(scraper, machine_id):
    print(f"🔁 Renewing machine {machine_id}...")

    data = {
        "month": "1",
        "submit": "1",
        "coupon_id": "0"
    }

    try:
        url = RENEW_URL.format(machine_id)
        response = scraper.post(url, data=data, headers=HEADERS)

        try:
            resp_json = response.json()
            msg = resp_json.get("msg", "")
        except Exception:
            msg = response.text

        if "请在到期前3天后再续费" in msg:
            print(f"⚠️  Machine {machine_id}: {msg}")
        elif "续费成功" in msg:
            print(f"✅  Machine {machine_id}: {msg}")
        else:
            print(f"❌  Machine {machine_id}: Unknown response: {msg}")
    except Exception as e:
        print(f"❌ Renew failed for machine {machine_id}: {e}")

def main():
    config = os.getenv("FC_PROFILES")
    if not config:
        print("❌ FC_PROFILES env variable not set.")
        sys.exit(1)

    try:
        profiles = json.loads(config)
    except Exception as e:
        print(f"❌ Failed to parse FC_PROFILES: {e}")
        sys.exit(1)

    if not isinstance(profiles, list):
        profiles = [profiles]

    for profile in profiles:
        username = profile.get("username")
        password = profile.get("password")
        machines = profile.get("machines", [])

        if not username or not password or not machines:
            print("⚠️ Skipping invalid profile.")
            continue

        scraper = cloudscraper.create_scraper()
        if login(scraper, username, password):
            for mid in machines:
                renew_machine(scraper, mid)
                time.sleep(1)

if __name__ == "__main__":
    main()

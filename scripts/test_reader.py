import requests
import json
import sseclient
import time

BASE_URL = "http://localhost/api/v1"

def run_test():
    print("🚀 Registering reader test user...")
    try:
        res = requests.post(f"{BASE_URL}/auth/register", json={
            "name": "Reader Tester",
            "email": "reader@example.com",
            "password": "password123"
        })
        print(res.json())
    except Exception as e:
        pass
    
    print("🚀 Logging in...")
    res = requests.post(f"{BASE_URL}/auth/login", json={
        "email": "reader@example.com",
        "password": "password123"
    })
    token = res.json()["data"]["token"]
    headers = {"Authorization": f"Bearer {token}"}

    print("🚀 Triggering Analysis for samber/lo...")
    response = requests.post(
        f"{BASE_URL}/stream/analyze",
        json={"source_url": "https://github.com/samber/lo"},
        headers=headers,
        stream=True
    )
    
    client = sseclient.SSEClient(response)
    project_id = None
    series_title = None
    chapters = []
    
    print("⏳ Listening to Analysis Stream...")
    for event in client.events():
        if event.event == "error":
            print(f"❌ Error: {event.data}")
            return
        elif event.event == "done":
            print("✅ Analysis Done!")
            data = json.loads(event.data)
            project_id = data.get("id")
            series_title = data.get("title")
            chapters = data.get("chapters", [])
            break
        elif event.event in ["chunk_analyzing", "chunk_done", "chunk_failed"]:
            print(f"[{event.event}] {event.data}")
        elif event.event == "analysis_step":
            print(f"🔄 Step: {event.data}")
    
    if not project_id:
        print("❌ Failed to get project_id")
        return
        
    print(f"\n✅ Created Series: {series_title} with {len(chapters)} chapters.")
    
    # Let's generate the first chapter to save time instead of all
    if not chapters:
        print("❌ No chapters generated.")
        return
        
    first_chapter = chapters[0]
    print(f"🚀 Generating First Chapter: {first_chapter['title']} (ID: {first_chapter['id']})")
    
    response = requests.post(
        f"{BASE_URL}/stream/generate",
        json={"blog_id": first_chapter["id"]},
        headers=headers,
        stream=True
    )
    
    client = sseclient.SSEClient(response)
    content = ""
    print("⏳ Listening to Generation Stream...")
    for event in client.events():
        if event.event == "error":
            print(f"❌ Error: {event.data}")
            break
        elif event.event == "done":
            print("\n✅ Generation Done!")
            break
        elif event.data:
            try:
                data = json.loads(event.data)
                if data.get("content"):
                    content += data["content"]
                    print(data["content"], end="", flush=True)
            except:
                pass
                
    print("\n\n✅ First Chapter Evaluation Content:")
    print("="*50)
    print(content[:500] + "...\n[TRUNCATED FOR DISPLAY]\n..." + content[-500:])
    print("="*50)
    
if __name__ == "__main__":
    run_test()

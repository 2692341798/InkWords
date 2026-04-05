import os
import time
from playwright.sync_api import sync_playwright, TimeoutError

def run_test():
    os.makedirs('test_results', exist_ok=True)
    
    console_logs = []
    network_logs = []
    
    def log_console(msg):
        log_line = f"CONSOLE [{msg.type}]: {msg.text}"
        console_logs.append(log_line)
        print(log_line)
        
    def log_request(req):
        if '/api/' in req.url:
            log_line = f"REQUEST  -> {req.method} {req.url}"
            network_logs.append(log_line)
            print(log_line)
            
    def log_response(res):
        if '/api/' in res.url:
            log_line = f"RESPONSE <- {res.status} {res.url}"
            network_logs.append(log_line)
            print(log_line)

    with sync_playwright() as p:
        browser = p.chromium.launch(headless=True)
        page = browser.new_page(viewport={'width': 1440, 'height': 900})
        
        page.on("console", log_console)
        page.on("request", log_request)
        page.on("response", log_response)
        
        try:
            print("Step 1: Navigating to App")
            page.goto('http://localhost:5173')
            page.wait_for_load_state('networkidle')
            page.screenshot(path='test_results/01_login_page.png')
            
            print("Step 2: Registering a new test user")
            # Click 立即注册 (index 4 from previous test)
            page.click('text=立即注册')
            page.wait_for_timeout(500)
            
            test_email = f"test_{int(time.time())}@example.com"
            page.fill('input[name="name"]', "Test User")
            page.fill('input[name="email"]', test_email)
            page.fill('input[name="password"]', "password123")
            page.screenshot(path='test_results/02_register_form.png')
            
            page.click('button[type="submit"]') # The 注册 button
            print(f"Submitted registration for {test_email}")
            
            print("Step 3: Waiting for Dashboard to load")
            # Wait for "拖拽或点击上传文件" or the Git url input
            page.wait_for_selector('text=拖拽或点击上传文件', timeout=15000)
            page.screenshot(path='test_results/03_dashboard.png')
            
            print("Step 4: Uploading backend/sample.pdf")
            # Assuming sample.pdf is in the backend directory
            page.locator('input[type="file"]').set_input_files('backend/sample.pdf')
            
            print("Step 5: Waiting for Analysis to complete")
            # It should eventually show "文件解析成功" and a "开始生成" button
            page.wait_for_selector('button:has-text("开始生成")', timeout=60000)
            page.screenshot(path='test_results/04_analysis_complete.png')
            
            print("Step 6: Starting generation")
            page.click('button:has-text("开始生成")')
            
            print("Step 7: Waiting for Generation to complete")
            # Wait for the generation to finish, the UI resets to the upload state
            page.wait_for_selector('text=拖拽或点击上传文件', timeout=180000)
            page.screenshot(path='test_results/05_generation_complete.png')
            
            print("Step 8: Checking History Blogs in Sidebar")
            # Click on the newly generated blog in the sidebar. It should be the first BookOpen item
            # Let's just click the first item in the sidebar that looks like a blog title.
            # wait for networkidle first
            page.wait_for_load_state('networkidle')
            
            # Click on the first element containing a typical title, or just use locator for the sidebar item.
            # We can find it by looking for the .truncate class inside the sidebar.
            first_blog = page.locator('.truncate').nth(0)
            if page.locator('.truncate').count() > 0:
                print(f"Clicking on blog: {first_blog.text_content()}")
                first_blog.click()
                page.wait_for_timeout(2000) # wait for editor to load
                page.screenshot(path='test_results/06_editor_view.png')
            else:
                print("No blog found in sidebar to click.")
                
            print("Test Completed Successfully!")
            
        except Exception as e:
            print(f"Test failed: {e}")
            page.screenshot(path='test_results/error_state.png')
        finally:
            browser.close()
            
    # Save logs
    with open('test_results/console.log', 'w') as f:
        f.write('\n'.join(console_logs))
    with open('test_results/network.log', 'w') as f:
        f.write('\n'.join(network_logs))
    print("Logs saved to test_results/")

if __name__ == "__main__":
    run_test()

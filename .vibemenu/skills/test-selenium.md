# Testing: Selenium WebDriver Skill Guide

## Overview

Selenium WebDriver setup, Page Object Model, explicit waits, screenshot on failure, and Selenium Grid for cross-browser parallel testing.

## Setup (Python)

```python
# requirements.txt
# selenium==4.18.1
# webdriver-manager==4.0.1

from selenium import webdriver
from selenium.webdriver.chrome.service import Service
from selenium.webdriver.chrome.options import Options
from webdriver_manager.chrome import ChromeDriverManager

def create_driver(headless: bool = True) -> webdriver.Chrome:
    options = Options()
    if headless:
        options.add_argument("--headless=new")
    options.add_argument("--no-sandbox")
    options.add_argument("--disable-dev-shm-usage")
    options.add_argument("--disable-gpu")
    options.add_argument("--window-size=1920,1080")

    service = Service(ChromeDriverManager().install())
    return webdriver.Chrome(service=service, options=options)
```

## Setup (Java)

```java
import io.github.bonigarcia.wdm.WebDriverManager;
import org.openqa.selenium.WebDriver;
import org.openqa.selenium.chrome.ChromeDriver;
import org.openqa.selenium.chrome.ChromeOptions;

public static WebDriver createDriver() {
    WebDriverManager.chromedriver().setup();

    ChromeOptions options = new ChromeOptions();
    options.addArguments("--headless=new", "--no-sandbox",
        "--disable-dev-shm-usage", "--window-size=1920,1080");

    return new ChromeDriver(options);
}
```

## Page Object Model (Python)

```python
from selenium.webdriver.common.by import By
from selenium.webdriver.support.ui import WebDriverWait
from selenium.webdriver.support import expected_conditions as EC
from selenium.webdriver.remote.webdriver import WebDriver


class LoginPage:
    URL = "/login"

    def __init__(self, driver: WebDriver, base_url: str):
        self._driver = driver
        self._base_url = base_url
        self._wait = WebDriverWait(driver, timeout=10)

    # Properties return locators lazily — driver finds them fresh each call
    @property
    def email_input(self):
        return self._wait.until(EC.visibility_of_element_located((By.CSS_SELECTOR, "[data-cy='email']")))

    @property
    def password_input(self):
        return self._wait.until(EC.visibility_of_element_located((By.CSS_SELECTOR, "[data-cy='password']")))

    @property
    def submit_button(self):
        return self._wait.until(EC.element_to_be_clickable((By.CSS_SELECTOR, "[data-cy='submit']")))

    @property
    def error_message(self):
        return self._wait.until(EC.visibility_of_element_located((By.CSS_SELECTOR, "[data-cy='error']")))

    def navigate(self):
        self._driver.get(self._base_url + self.URL)
        return self

    def login(self, email: str, password: str) -> "DashboardPage":
        self.email_input.clear()
        self.email_input.send_keys(email)
        self.password_input.clear()
        self.password_input.send_keys(password)
        self.submit_button.click()
        return DashboardPage(self._driver, self._base_url)


class DashboardPage:
    def __init__(self, driver: WebDriver, base_url: str):
        self._driver = driver
        self._base_url = base_url
        self._wait = WebDriverWait(driver, timeout=10)

    @property
    def welcome_heading(self):
        return self._wait.until(EC.visibility_of_element_located((By.CSS_SELECTOR, "h1")))

    def is_loaded(self) -> bool:
        return "/dashboard" in self._driver.current_url
```

## Test with Explicit Waits

```python
import pytest
from selenium.webdriver.support.ui import WebDriverWait
from selenium.webdriver.support import expected_conditions as EC


@pytest.fixture(scope="module")
def driver():
    drv = create_driver(headless=True)
    yield drv
    drv.quit()


class TestLogin:
    def test_successful_login(self, driver):
        login_page = LoginPage(driver, base_url="http://localhost:3000")
        dashboard = login_page.navigate().login("alice@example.com", "password")

        assert dashboard.is_loaded(), "Expected to land on /dashboard"
        assert "Welcome" in dashboard.welcome_heading.text

    def test_invalid_credentials(self, driver):
        login_page = LoginPage(driver, base_url="http://localhost:3000").navigate()
        login_page.login("bad@example.com", "wrong")

        error = login_page.error_message
        assert "Invalid credentials" in error.text

    def test_successful_login_java_style(self, driver):
        try:
            login_page = LoginPage(driver, base_url="http://localhost:3000")
            login_page.navigate().login("alice@example.com", "password")

            wait = WebDriverWait(driver, 10)
            wait.until(EC.url_contains("/dashboard"))
        except Exception as e:
            # Screenshot on failure
            driver.save_screenshot(f"failure_{self.__class__.__name__}.png")
            raise
```

## Explicit vs Implicit Waits

```python
# GOOD: Explicit wait — precise and readable
wait = WebDriverWait(driver, timeout=10, poll_frequency=0.5)
element = wait.until(EC.element_to_be_clickable((By.ID, "submit")))

# GOOD: Expected conditions
EC.visibility_of_element_located((By.CSS_SELECTOR, ".modal"))
EC.invisibility_of_element((By.CSS_SELECTOR, ".spinner"))
EC.text_to_be_present_in_element((By.ID, "status"), "completed")
EC.staleness_of(old_element)  # wait for element to be replaced

# BAD: Implicit wait — unreliable, interacts badly with explicit waits
# driver.implicitly_wait(10)  # AVOID
```

## Screenshot on Failure (pytest fixture)

```python
# conftest.py
import pytest

@pytest.hookimpl(tryfirst=True, hookwrapper=True)
def pytest_runtest_makereport(item, call):
    outcome = yield
    report = outcome.get_result()
    if report.when == "call" and report.failed:
        driver = item.funcargs.get("driver")
        if driver:
            path = f"screenshots/{item.nodeid.replace('/', '_').replace('::', '_')}.png"
            os.makedirs("screenshots", exist_ok=True)
            driver.save_screenshot(path)
```

## Selenium Grid (Parallel Cross-Browser)

```yaml
# docker-compose.selenium.yml
services:
  hub:
    image: selenium/hub:4.18
    ports:
      - "4442:4442"
      - "4443:4443"
      - "4444:4444"

  chrome:
    image: selenium/node-chrome:4.18
    shm_size: 2g
    depends_on: [hub]
    environment:
      SE_EVENT_BUS_HOST: hub
      SE_EVENT_BUS_PUBLISH_PORT: 4442
      SE_EVENT_BUS_SUBSCRIBE_PORT: 4443
      SE_NODE_MAX_SESSIONS: 3

  firefox:
    image: selenium/node-firefox:4.18
    shm_size: 2g
    depends_on: [hub]
    environment:
      SE_EVENT_BUS_HOST: hub
      SE_EVENT_BUS_PUBLISH_PORT: 4442
      SE_EVENT_BUS_SUBSCRIBE_PORT: 4443
```

```python
# Remote WebDriver for Grid
from selenium.webdriver.remote.webdriver import WebDriver as RemoteDriver

def create_remote_driver(browser: str = "chrome") -> RemoteDriver:
    options = ChromeOptions() if browser == "chrome" else FirefoxOptions()
    return webdriver.Remote(
        command_executor="http://hub:4444/wd/hub",
        options=options,
    )
```

```java
// Java Grid connection
ChromeOptions options = new ChromeOptions();
WebDriver driver = new RemoteWebDriver(
    new URL("http://hub:4444/wd/hub"), options);
```

## Key Rules

- Always use explicit waits (`WebDriverWait` + `ExpectedConditions`) — never implicit waits.
- Page Object Model is mandatory — no `driver.find_element` calls in test methods.
- Use `@property` methods in Python POMs so elements are re-fetched each access (avoids stale element).
- Take a screenshot on every test failure and attach to CI artifacts.
- Run `driver.quit()` in fixture teardown (`yield` pattern) — not in try/finally inside tests.
- For Selenium Grid: set `shm_size: 2g` on browser nodes to prevent Chrome crashes.
- Prefer CSS selectors with `data-testid` or `data-cy` attributes — avoid XPath unless necessary.

# Kotlin Multiplatform + Compose Multiplatform Skill Guide

## Project Layout

```
project/
├── gradle/
├── settings.gradle.kts
├── composeApp/                   # Compose Multiplatform UI module
│   ├── build.gradle.kts
│   └── src/
│       ├── commonMain/           # Shared UI code
│       │   ├── kotlin/com/example/
│       │   │   ├── App.kt        # Root composable
│       │   │   ├── screens/
│       │   │   └── components/
│       │   └── composeResources/ # Shared images, strings
│       ├── androidMain/          # Android-specific
│       │   └── kotlin/.../MainActivity.kt
│       └── iosMain/              # iOS-specific
│           └── kotlin/.../MainViewController.kt
├── shared/                       # KMP business logic module
│   ├── build.gradle.kts
│   └── src/
│       ├── commonMain/kotlin/
│       │   ├── data/
│       │   │   ├── model/
│       │   │   └── repository/
│       │   ├── domain/
│       │   │   └── usecase/
│       │   └── presentation/
│       │       └── viewmodel/
│       ├── androidMain/kotlin/   # Android actual implementations
│       ├── iosMain/kotlin/       # iOS actual implementations
│       └── desktopMain/kotlin/   # Desktop actual implementations
└── iosApp/                       # Xcode project entry point
```

## Key Dependencies (shared/build.gradle.kts)

```kotlin
kotlin {
    androidTarget()
    iosX64(); iosArm64(); iosSimulatorArm64()
    jvm("desktop")

    sourceSets {
        commonMain.dependencies {
            implementation("org.jetbrains.kotlinx:kotlinx-coroutines-core:1.9.0")
            implementation("org.jetbrains.kotlinx:kotlinx-serialization-json:1.7.3")
            implementation("io.ktor:ktor-client-core:3.0.1")
            implementation("io.ktor:ktor-client-content-negotiation:3.0.1")
            implementation("io.ktor:ktor-serialization-kotlinx-json:3.0.1")
        }
        androidMain.dependencies {
            implementation("io.ktor:ktor-client-okhttp:3.0.1")
        }
        iosMain.dependencies {
            implementation("io.ktor:ktor-client-darwin:3.0.1")
        }
        val desktopMain by getting {
            dependencies { implementation("io.ktor:ktor-client-okhttp:3.0.1") }
        }
    }
}
```

## expect / actual Pattern

```kotlin
// commonMain — declare the interface
expect fun getPlatformName(): String
expect class DatabaseDriver(context: Any? = null) {
    fun createDriver(): SqlDriver
}

// androidMain — Android implementation
actual fun getPlatformName(): String = "Android ${android.os.Build.VERSION.RELEASE}"
actual class DatabaseDriver actual constructor(context: Any?) {
    actual fun createDriver(): SqlDriver =
        AndroidSqliteDriver(AppDatabase.Schema, context as Context, "app.db")
}

// iosMain — iOS implementation
actual fun getPlatformName(): String = UIDevice.currentDevice.systemName()
actual class DatabaseDriver actual constructor(context: Any?) {
    actual fun createDriver(): SqlDriver =
        NativeSqliteDriver(AppDatabase.Schema, "app.db")
}
```

## Shared ViewModel (commonMain)

```kotlin
// shared/src/commonMain/kotlin/presentation/viewmodel/UsersViewModel.kt
import kotlinx.coroutines.flow.*
import kotlinx.coroutines.launch

class UsersViewModel(
    private val repository: UserRepository,
) : CoroutineViewModel() {

    private val _uiState = MutableStateFlow<UsersUiState>(UsersUiState.Loading)
    val uiState: StateFlow<UsersUiState> = _uiState.asStateFlow()

    init { loadUsers() }

    fun loadUsers() {
        viewModelScope.launch {
            _uiState.value = UsersUiState.Loading
            try {
                repository.getUsers().collect { users ->
                    _uiState.value = UsersUiState.Success(users)
                }
            } catch (e: Exception) {
                _uiState.value = UsersUiState.Error(e.message ?: "Unknown error")
            }
        }
    }
}

sealed interface UsersUiState {
    data object Loading : UsersUiState
    data class Success(val users: List<User>) : UsersUiState
    data class Error(val message: String) : UsersUiState
}

// CoroutineViewModel base — common across platforms
abstract class CoroutineViewModel {
    protected val viewModelScope = CoroutineScope(SupervisorJob() + Dispatchers.Main)
    fun clear() { viewModelScope.cancel() }
}
```

## Shared UI in commonMain (Compose Multiplatform)

```kotlin
// composeApp/src/commonMain/kotlin/App.kt
import androidx.compose.runtime.*
import androidx.compose.material3.*

@Composable
fun App() {
    MaterialTheme {
        val viewModel = remember { UsersViewModel(UserRepositoryImpl()) }
        val uiState by viewModel.uiState.collectAsState()

        when (val state = uiState) {
            is UsersUiState.Loading -> CircularProgressIndicator()
            is UsersUiState.Success -> UserList(users = state.users)
            is UsersUiState.Error -> Text("Error: ${state.message}")
        }
    }
}

@Composable
fun UserList(users: List<User>) {
    LazyColumn {
        items(users, key = { it.id }) { user ->
            ListItem(
                headlineContent = { Text(user.name) },
                supportingContent = { Text(user.email) },
            )
        }
    }
}
```

## Android Entry Point

```kotlin
// composeApp/src/androidMain/kotlin/MainActivity.kt
class MainActivity : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContent { App() }
    }
}
```

## iOS Entry Point

```kotlin
// composeApp/src/iosMain/kotlin/MainViewController.kt
import androidx.compose.ui.window.ComposeUIViewController
import platform.UIKit.UIViewController

fun MainViewController(): UIViewController = ComposeUIViewController { App() }
```

## Platform-Specific Navigation

```kotlin
// commonMain — expect navigation interface
expect class AppNavigator {
    fun navigateTo(route: String)
    fun goBack()
}

// androidMain — NavController-based
actual class AppNavigator actual constructor(
    private val navController: NavController,
) {
    actual fun navigateTo(route: String) { navController.navigate(route) }
    actual fun goBack() { navController.popBackStack() }
}

// iosMain — NavigationStack-based (via UIKit)
actual class AppNavigator {
    private val navigationController = UINavigationController()
    actual fun navigateTo(route: String) { /* push ViewController */ }
    actual fun goBack() { navigationController.popViewControllerAnimated(true) }
}
```

## SharedFlow for One-Shot Events

```kotlin
// In ViewModel
private val _events = MutableSharedFlow<UserEvent>()
val events: SharedFlow<UserEvent> = _events.asSharedFlow()

fun deleteUser(id: String) {
    viewModelScope.launch {
        repository.deleteUser(id)
        _events.emit(UserEvent.Deleted(id))
    }
}

sealed interface UserEvent {
    data class Deleted(val id: String) : UserEvent
}

// In Composable
LaunchedEffect(Unit) {
    viewModel.events.collect { event ->
        when (event) {
            is UserEvent.Deleted -> showSnackbar("User deleted")
        }
    }
}
```

## Key Rules

- Business logic lives in `commonMain` — zero platform-specific code in ViewModel/Repository.
- Use `expect`/`actual` only for platform-specific capabilities (DB drivers, platform APIs, filesystem).
- Use `StateFlow` for UI state, `SharedFlow` for one-shot events (navigation, toasts).
- `viewModelScope` must use `Dispatchers.Main` in commonMain — each platform provides the correct Main dispatcher.
- Call `viewModel.clear()` when the iOS UIViewController disappears (use `DisposableEffect` in Compose or `onDestroy`).
- Compose Multiplatform handles Android/iOS/Desktop UI from a single `@Composable` in commonMain.
- Use `collectAsStateWithLifecycle()` on Android; `collectAsState()` is safe on all platforms.
- Shared resources (images, strings) go in `composeResources/` — access via generated `Res` class.

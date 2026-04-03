# Jetpack Compose Skill Guide

## Project Layout

```
app/
├── build.gradle.kts
├── src/main/
│   ├── AndroidManifest.xml
│   └── java/com/example/app/
│       ├── MainActivity.kt
│       ├── ui/
│       │   ├── theme/
│       │   │   ├── Theme.kt
│       │   │   ├── Color.kt
│       │   │   └── Type.kt
│       │   └── screens/
│       │       ├── UserListScreen.kt
│       │       └── UserDetailScreen.kt
│       ├── viewmodel/
│       │   └── UsersViewModel.kt
│       ├── data/
│       │   ├── model/
│       │   └── repository/
│       └── navigation/
│           └── AppNavGraph.kt
```

## Key Dependencies (build.gradle.kts)

```kotlin
dependencies {
    implementation(platform("androidx.compose:compose-bom:2024.11.00"))
    implementation("androidx.compose.ui:ui")
    implementation("androidx.compose.ui:ui-tooling-preview")
    implementation("androidx.compose.material3:material3")
    implementation("androidx.activity:activity-compose:1.9.3")
    implementation("androidx.lifecycle:lifecycle-viewmodel-compose:2.8.7")
    implementation("androidx.navigation:navigation-compose:2.8.4")
    implementation("androidx.lifecycle:lifecycle-runtime-compose:2.8.7")
    debugImplementation("androidx.compose.ui:ui-tooling")
}
```

## @Composable Basics

```kotlin
@Composable
fun UserCard(user: User, onClick: () -> Unit) {
    Card(
        modifier = Modifier
            .fillMaxWidth()
            .padding(8.dp)
            .clickable(onClick = onClick),
        elevation = CardDefaults.cardElevation(defaultElevation = 2.dp),
    ) {
        Column(modifier = Modifier.padding(16.dp)) {
            Text(text = user.name, style = MaterialTheme.typography.titleMedium)
            Text(text = user.email, style = MaterialTheme.typography.bodyMedium)
        }
    }
}

@Preview(showBackground = true)
@Composable
fun UserCardPreview() {
    AppTheme { UserCard(user = User("1", "Alice", "alice@example.com"), onClick = {}) }
}
```

## State: remember / rememberSaveable

```kotlin
@Composable
fun Counter() {
    // Lost on recomposition if not remembered; lost on config change
    var count by remember { mutableStateOf(0) }

    // Survives config changes (rotation) via SavedState
    var input by rememberSaveable { mutableStateOf("") }

    Column {
        Text("Count: $count")
        Button(onClick = { count++ }) { Text("Increment") }
        OutlinedTextField(
            value = input,
            onValueChange = { input = it },
            label = { Text("Name") }
        )
    }
}
```

## State Hoisting

```kotlin
// Hoisted: parent owns state, child is stateless
@Composable
fun SearchBar(
    query: String,                     // value down
    onQueryChange: (String) -> Unit,   // event up
    modifier: Modifier = Modifier,
) {
    OutlinedTextField(
        value = query,
        onValueChange = onQueryChange,
        modifier = modifier.fillMaxWidth(),
        label = { Text("Search") },
    )
}

@Composable
fun SearchScreen() {
    var query by remember { mutableStateOf("") }
    SearchBar(query = query, onQueryChange = { query = it })
}
```

## ViewModel + StateFlow

```kotlin
// UsersViewModel.kt
@HiltViewModel
class UsersViewModel @Inject constructor(
    private val repository: UserRepository,
) : ViewModel() {
    private val _uiState = MutableStateFlow<UsersUiState>(UsersUiState.Loading)
    val uiState: StateFlow<UsersUiState> = _uiState.asStateFlow()

    init { loadUsers() }

    fun loadUsers() {
        viewModelScope.launch {
            _uiState.value = UsersUiState.Loading
            repository.getUsers()
                .onSuccess { _uiState.value = UsersUiState.Success(it) }
                .onFailure { _uiState.value = UsersUiState.Error(it.message ?: "Error") }
        }
    }
}

sealed interface UsersUiState {
    data object Loading : UsersUiState
    data class Success(val users: List<User>) : UsersUiState
    data class Error(val message: String) : UsersUiState
}
```

```kotlin
// UsersScreen.kt
@Composable
fun UsersScreen(viewModel: UsersViewModel = hiltViewModel()) {
    val uiState by viewModel.uiState.collectAsStateWithLifecycle()

    when (val state = uiState) {
        is UsersUiState.Loading -> CircularProgressIndicator()
        is UsersUiState.Success -> UserList(users = state.users)
        is UsersUiState.Error -> ErrorMessage(message = state.message, onRetry = viewModel::loadUsers)
    }
}
```

## Layout: Column / Row / Box / Modifier

```kotlin
@Composable
fun ProfileHeader(user: User) {
    Box(modifier = Modifier.fillMaxWidth()) {
        // Background
        Image(
            painter = painterResource(R.drawable.hero),
            contentDescription = null,
            modifier = Modifier.fillMaxWidth().height(200.dp),
            contentScale = ContentScale.Crop,
        )
        // Overlay in bottom-start corner
        Row(
            modifier = Modifier
                .align(Alignment.BottomStart)
                .padding(16.dp),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(8.dp),
        ) {
            AsyncImage(model = user.avatar, contentDescription = user.name,
                modifier = Modifier.size(48.dp).clip(CircleShape))
            Text(user.name, color = Color.White, style = MaterialTheme.typography.titleLarge)
        }
    }
}
```

## LazyColumn / LazyRow

```kotlin
@Composable
fun UserList(users: List<User>, onUserClick: (User) -> Unit) {
    LazyColumn(
        verticalArrangement = Arrangement.spacedBy(8.dp),
        contentPadding = PaddingValues(16.dp),
    ) {
        items(users, key = { it.id }) { user ->
            UserCard(user = user, onClick = { onUserClick(user) })
        }
    }
}
```

## NavHost

```kotlin
// navigation/AppNavGraph.kt
@Composable
fun AppNavGraph(navController: NavHostController) {
    NavHost(navController = navController, startDestination = "users") {
        composable("users") {
            UsersScreen(onUserClick = { id -> navController.navigate("users/$id") })
        }
        composable(
            "users/{id}",
            arguments = listOf(navArgument("id") { type = NavType.StringType }),
        ) { backStack ->
            val id = backStack.arguments?.getString("id") ?: return@composable
            UserDetailScreen(userId = id, onBack = { navController.popBackStack() })
        }
    }
}

// MainActivity.kt
val navController = rememberNavController()
AppNavGraph(navController = navController)
```

## LaunchedEffect

```kotlin
@Composable
fun AutoRefreshScreen(interval: Long = 30_000L) {
    val viewModel: UsersViewModel = hiltViewModel()

    // Runs coroutine in composition scope; re-launches if key changes
    LaunchedEffect(Unit) {
        while (true) {
            viewModel.loadUsers()
            delay(interval)
        }
    }
    // ...
}

// One-shot effect triggered by a key
LaunchedEffect(userId) {
    viewModel.loadUser(userId)
}
```

## Material 3 Theming

```kotlin
// ui/theme/Theme.kt
@Composable
fun AppTheme(content: @Composable () -> Unit) {
    MaterialTheme(
        colorScheme = lightColorScheme(
            primary = Blue40,
            secondary = BlueGrey40,
        ),
        typography = AppTypography,
        content = content,
    )
}

// Usage in composable
MaterialTheme.colorScheme.primary
MaterialTheme.typography.bodyLarge
```

## Key Rules

- Every `@Composable` function name starts with an uppercase letter; follows composition over inheritance.
- Hoist state to the lowest common ancestor that needs it.
- Use `rememberSaveable` for UI state that must survive rotation/process death.
- Collect `StateFlow` with `collectAsStateWithLifecycle()` (not `collectAsState`) to respect lifecycle.
- Use `key = { item.id }` in `LazyColumn` items to preserve item identity during reordering.
- Never call `remember` conditionally — it must run on every composition.
- `LaunchedEffect(key)` relaunches when the key changes; `DisposableEffect` for cleanup.
- All colors and typography must come from `MaterialTheme` — never hardcode.

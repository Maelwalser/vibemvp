# UIKit Skill Guide

## Project Layout

```
App/
├── App.xcodeproj
└── App/
    ├── AppDelegate.swift
    ├── SceneDelegate.swift
    ├── Features/
    │   └── Users/
    │       ├── UsersViewController.swift
    │       ├── UserDetailViewController.swift
    │       ├── UserCell.swift
    │       └── UsersViewModel.swift
    ├── Shared/
    │   ├── Base/
    │   │   └── BaseViewController.swift
    │   ├── Extensions/
    │   └── Services/
    └── Resources/
        └── Main.storyboard   # or use programmatic UI
```

## UIViewController Lifecycle

```swift
class UsersViewController: UIViewController {

    // MARK: - Properties
    private let viewModel: UsersViewModel
    private lazy var tableView = UITableView()

    // MARK: - Init
    init(viewModel: UsersViewModel) {
        self.viewModel = viewModel
        super.init(nibName: nil, bundle: nil)
    }

    required init?(coder: NSCoder) { fatalError("init(coder:) not supported") }

    // MARK: - Lifecycle

    override func viewDidLoad() {
        super.viewDidLoad()
        // Called ONCE after view is in memory — set up UI and bindings here
        setupUI()
        setupConstraints()
        bindViewModel()
        viewModel.loadUsers()
    }

    override func viewWillAppear(_ animated: Bool) {
        super.viewWillAppear(animated)
        // Called each time the view appears — good for refresh
    }

    override func viewDidAppear(_ animated: Bool) {
        super.viewDidAppear(animated)
        // View is visible — start animations / analytics
    }

    override func viewWillDisappear(_ animated: Bool) {
        super.viewWillDisappear(animated)
        // About to leave — pause tasks
    }

    deinit {
        // Clean up observers / timers
        NotificationCenter.default.removeObserver(self)
    }
}
```

## Programmatic Auto Layout

```swift
private func setupUI() {
    view.backgroundColor = .systemBackground
    title = "Users"
    navigationItem.rightBarButtonItem = UIBarButtonItem(
        systemItem: .add, primaryAction: UIAction { [weak self] _ in
            self?.showAddUser()
        }
    )

    tableView.translatesAutoresizingMaskIntoConstraints = false
    tableView.register(UserCell.self, forCellReuseIdentifier: UserCell.reuseId)
    tableView.dataSource = self
    tableView.delegate = self
    view.addSubview(tableView)
}

private func setupConstraints() {
    NSLayoutConstraint.activate([
        tableView.topAnchor.constraint(equalTo: view.safeAreaLayoutGuide.topAnchor),
        tableView.leadingAnchor.constraint(equalTo: view.leadingAnchor),
        tableView.trailingAnchor.constraint(equalTo: view.trailingAnchor),
        tableView.bottomAnchor.constraint(equalTo: view.bottomAnchor),
    ])
}
```

## UIStackView

```swift
let stack = UIStackView(arrangedSubviews: [titleLabel, subtitleLabel, actionButton])
stack.axis = .vertical
stack.spacing = 8
stack.alignment = .leading
stack.distribution = .fill
stack.translatesAutoresizingMaskIntoConstraints = false
containerView.addSubview(stack)
```

## UITableView (DataSource + Delegate)

```swift
extension UsersViewController: UITableViewDataSource {
    func tableView(_ tableView: UITableView, numberOfRowsInSection section: Int) -> Int {
        viewModel.users.count
    }

    func tableView(_ tableView: UITableView, cellForRowAt indexPath: IndexPath) -> UITableViewCell {
        let cell = tableView.dequeueReusableCell(
            withIdentifier: UserCell.reuseId, for: indexPath
        ) as! UserCell
        cell.configure(with: viewModel.users[indexPath.row])
        return cell
    }

    func tableView(_ tableView: UITableView,
                   commit editingStyle: UITableViewCell.EditingStyle,
                   forRowAt indexPath: IndexPath) {
        if editingStyle == .delete {
            viewModel.deleteUser(at: indexPath.row)
            tableView.deleteRows(at: [indexPath], with: .automatic)
        }
    }
}

extension UsersViewController: UITableViewDelegate {
    func tableView(_ tableView: UITableView, didSelectRowAt indexPath: IndexPath) {
        tableView.deselectRow(at: indexPath, animated: true)
        let user = viewModel.users[indexPath.row]
        let detail = UserDetailViewController(user: user)
        navigationController?.pushViewController(detail, animated: true)
    }
}
```

## Custom UITableViewCell

```swift
final class UserCell: UITableViewCell {
    static let reuseId = "UserCell"

    private let nameLabel = UILabel()
    private let emailLabel = UILabel()

    override init(style: UITableViewCell.CellStyle, reuseIdentifier: String?) {
        super.init(style: style, reuseIdentifier: reuseIdentifier)
        setup()
    }

    required init?(coder: NSCoder) { fatalError() }

    private func setup() {
        nameLabel.font = .preferredFont(forTextStyle: .headline)
        emailLabel.font = .preferredFont(forTextStyle: .subheadline)
        emailLabel.textColor = .secondaryLabel

        let stack = UIStackView(arrangedSubviews: [nameLabel, emailLabel])
        stack.axis = .vertical
        stack.spacing = 4
        stack.translatesAutoresizingMaskIntoConstraints = false
        contentView.addSubview(stack)
        NSLayoutConstraint.activate([
            stack.topAnchor.constraint(equalTo: contentView.topAnchor, constant: 12),
            stack.leadingAnchor.constraint(equalTo: contentView.leadingAnchor, constant: 16),
            stack.trailingAnchor.constraint(equalTo: contentView.trailingAnchor, constant: -16),
            stack.bottomAnchor.constraint(equalTo: contentView.bottomAnchor, constant: -12),
        ])
    }

    func configure(with user: User) {
        nameLabel.text = user.name
        emailLabel.text = user.email
    }
}
```

## UICollectionView (Modern Diffable)

```swift
// Modern diffable data source (iOS 14+)
enum Section { case main }
typealias DataSource = UICollectionViewDiffableDataSource<Section, User>

var dataSource: DataSource!

private func setupCollectionView() {
    let config = UICollectionLayoutListConfiguration(appearance: .insetGrouped)
    let layout = UICollectionViewCompositionalLayout.list(using: config)
    collectionView = UICollectionView(frame: view.bounds, collectionViewLayout: layout)
    collectionView.autoresizingMask = [.flexibleWidth, .flexibleHeight]
    view.addSubview(collectionView)

    let cellReg = UICollectionView.CellRegistration<UICollectionViewListCell, User> { cell, _, user in
        var content = cell.defaultContentConfiguration()
        content.text = user.name
        content.secondaryText = user.email
        cell.contentConfiguration = content
    }

    dataSource = DataSource(collectionView: collectionView) { cv, indexPath, user in
        cv.dequeueConfiguredReusableCell(using: cellReg, for: indexPath, item: user)
    }
}

private func applySnapshot(users: [User]) {
    var snapshot = NSDiffableDataSourceSnapshot<Section, User>()
    snapshot.appendSections([.main])
    snapshot.appendItems(users)
    dataSource.apply(snapshot, animatingDifferences: true)
}
```

## Navigation: Push / Present / Dismiss

```swift
// Push (NavigationController)
let detailVC = UserDetailViewController(user: user)
navigationController?.pushViewController(detailVC, animated: true)

// Pop
navigationController?.popViewController(animated: true)
navigationController?.popToRootViewController(animated: true)

// Present modal
let newUserVC = NewUserViewController()
let nav = UINavigationController(rootViewController: newUserVC)
nav.modalPresentationStyle = .pageSheet
present(nav, animated: true)

// Dismiss modal
dismiss(animated: true)
```

## Target-Action

```swift
// Modern closure-based (iOS 14+)
let button = UIButton(type: .system)
button.setTitle("Add User", for: .normal)
button.addAction(UIAction { [weak self] _ in
    self?.showAddUser()
}, for: .touchUpInside)

// Legacy target-action
button.addTarget(self, action: #selector(handleAdd), for: .touchUpInside)

@objc private func handleAdd() {
    showAddUser()
}
```

## Combine Bindings (ViewModel → View)

```swift
import Combine

final class UsersViewModel {
    @Published var users: [User] = []
    @Published var isLoading = false
    private var cancellables = Set<AnyCancellable>()
    // ...
}

// In ViewController
private var cancellables = Set<AnyCancellable>()

private func bindViewModel() {
    viewModel.$users
        .receive(on: DispatchQueue.main)
        .sink { [weak self] _ in
            self?.tableView.reloadData()
        }
        .store(in: &cancellables)

    viewModel.$isLoading
        .receive(on: DispatchQueue.main)
        .sink { [weak self] loading in
            loading ? self?.activityIndicator.startAnimating()
                    : self?.activityIndicator.stopAnimating()
        }
        .store(in: &cancellables)
}
```

## UIKit vs SwiftUI

| Concern | UIKit | SwiftUI |
|---------|-------|---------|
| Paradigm | Imperative — mutate views | Declarative — describe view state |
| State | `@Published` + manual reload | `@State` / `@Observable` + auto re-render |
| Lists | `UITableView` dataSource + delegate | `List` / `LazyVStack` |
| Navigation | `UINavigationController` push/pop | `NavigationStack` + `navigationDestination` |
| Layout | Auto Layout constraints | VStack/HStack/ZStack |
| Lifecycle | `viewDidLoad` / `viewWillAppear` | `.onAppear` / `.task` |
| Complexity | Higher boilerplate | Lower boilerplate |
| Control | Fine-grained (custom animations, perf) | Less granular |

## Key Rules

- Always call `super` in lifecycle methods (`viewDidLoad`, `viewWillAppear`, etc.).
- Use `[weak self]` in closures that capture the view controller to prevent retain cycles.
- Register cells before the table view is shown; dequeue with typed `as!` cast.
- Use diffable data source (iOS 14+) instead of manual `insertRows`/`deleteRows`.
- Prefer `UIAction` closures over `#selector` for new button actions.
- Use `UIStackView` instead of manual constraint arithmetic for simple layouts.
- Never update UI from a background thread — use `DispatchQueue.main.async`.
- Set `translatesAutoresizingMaskIntoConstraints = false` on every programmatic view.

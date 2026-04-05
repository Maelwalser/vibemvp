# Python + Django REST Framework Skill Guide

## Project Layout

```
myproject/
├── manage.py
├── config/
│   ├── settings/
│   │   ├── base.py
│   │   ├── development.py
│   │   └── production.py
│   ├── urls.py
│   └── wsgi.py
├── apps/
│   └── users/
│       ├── models.py
│       ├── serializers.py
│       ├── views.py
│       ├── urls.py
│       └── tests/
└── pyproject.toml
```

## Dependencies

```toml
[project]
dependencies = [
    "django>=5.0",
    "djangorestframework>=3.15.0",
    "django-filter>=24.0",
    "psycopg[binary]>=3.1.0",
]
```

## ModelSerializer

```python
# apps/users/serializers.py
from rest_framework import serializers
from .models import User, Post

class UserSerializer(serializers.ModelSerializer):
    full_name = serializers.SerializerMethodField()

    class Meta:
        model = User
        fields = ["id", "email", "full_name", "created_at"]
        read_only_fields = ["id", "created_at"]

    def get_full_name(self, obj: User) -> str:
        return f"{obj.first_name} {obj.last_name}"

class PostSerializer(serializers.ModelSerializer):
    author = UserSerializer(read_only=True)
    author_id = serializers.PrimaryKeyRelatedField(
        queryset=User.objects.all(), source="author", write_only=True
    )

    class Meta:
        model = Post
        fields = ["id", "title", "body", "author", "author_id", "created_at"]
        read_only_fields = ["id", "created_at"]
```

## ModelViewSet

```python
# apps/users/views.py
from rest_framework import viewsets, permissions, status
from rest_framework.decorators import action
from rest_framework.request import Request
from rest_framework.response import Response
from django_filters.rest_framework import DjangoFilterBackend
from .models import User
from .serializers import UserSerializer

class UserViewSet(viewsets.ModelViewSet):
    queryset = User.objects.select_related("profile").prefetch_related("roles").all()
    serializer_class = UserSerializer
    permission_classes = [permissions.IsAuthenticated]
    filter_backends = [DjangoFilterBackend]
    filterset_fields = ["is_active"]
    pagination_class = None  # or assign a PageNumberPagination subclass

    def get_queryset(self):
        # Never use .all() without ordering — non-deterministic pagination
        return super().get_queryset().order_by("-created_at")

    @action(detail=True, methods=["post"], url_path="deactivate")
    def deactivate(self, request: Request, pk: int | None = None) -> Response:
        user = self.get_object()
        user.is_active = False
        user.save(update_fields=["is_active"])
        return Response({"status": "deactivated"}, status=status.HTTP_200_OK)
```

## Router Registration

```python
# apps/users/urls.py
from rest_framework.routers import DefaultRouter
from .views import UserViewSet

router = DefaultRouter()
router.register(r"users", UserViewSet, basename="user")

urlpatterns = router.urls
```

```python
# config/urls.py
from django.urls import path, include

urlpatterns = [
    path("api/v1/", include("apps.users.urls")),
]
```

## Permissions and Authentication

```python
# Custom permission
from rest_framework.permissions import BasePermission

class IsOwner(BasePermission):
    def has_object_permission(self, request, view, obj) -> bool:
        return obj.owner == request.user

# On ViewSet
class PostViewSet(viewsets.ModelViewSet):
    permission_classes = [permissions.IsAuthenticated, IsOwner]
    authentication_classes = [SessionAuthentication, TokenAuthentication]
```

## N+1 Prevention

```python
# ALWAYS use select_related (FK/OneToOne) and prefetch_related (M2M/reverse FK)
queryset = (
    Post.objects.select_related("author", "category")
    .prefetch_related("tags", "comments__author")
    .filter(published=True)
    .order_by("-published_at")
)
```

## Pagination

```python
# config/settings/base.py
REST_FRAMEWORK = {
    "DEFAULT_PAGINATION_CLASS": "rest_framework.pagination.PageNumberPagination",
    "PAGE_SIZE": 20,
    "DEFAULT_AUTHENTICATION_CLASSES": [
        "rest_framework.authentication.TokenAuthentication",
    ],
    "DEFAULT_PERMISSION_CLASSES": [
        "rest_framework.permissions.IsAuthenticated",
    ],
}
```

## Error Handling

- Raise `serializers.ValidationError` inside `validate_<field>()` or `validate()` for input errors.
- Raise `rest_framework.exceptions.NotFound`, `PermissionDenied`, etc. for HTTP semantics.
- Override `handle_exception()` on the view or use a custom exception handler in settings.

## Key Rules

- Always set `read_only_fields` on serializers — never expose PK/timestamps as writable.
- Use `select_related`/`prefetch_related` in `get_queryset()` — never inside serializers.
- Use `@action` for non-CRUD endpoints instead of standalone `APIView`.
- Register all ViewSets via `DefaultRouter` — do not manually wire URL patterns for standard CRUD.
- Add ordering to every queryset used with pagination to prevent non-deterministic pages.

from pydantic import BaseModel, EmailStr, SecretStr


class BasicUserSchema(BaseModel):
    id: int
    username: str
    email: EmailStr | None


class UserLogin(BaseModel):
    username: str
    password: SecretStr


class UserRegister(BaseModel):
    username: str
    email: EmailStr | None
    password: SecretStr


class TokenResponse(BaseModel):
    id: int
    username: str
    email: EmailStr | None
    token_type: str
    access_token: str


class UserChangePassword(BaseModel):
    username: str
    current_password: SecretStr
    new_password: SecretStr


class UserChange(BaseModel):
    username: str
    new_username: str | None
    new_email: EmailStr | None

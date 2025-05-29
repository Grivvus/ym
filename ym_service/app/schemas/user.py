from pydantic import BaseModel, EmailStr, SecretStr


class UserLogin(BaseModel):
    username: str
    password: SecretStr


class UserRegister(BaseModel):
    username: str
    email: EmailStr | None
    password: SecretStr


class TokenResponse(BaseModel):
    token_type: str
    access_token: str

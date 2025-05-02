from pydantic import BaseModel, EmailStr, SecretStr


class UserLogin(BaseModel):
    username: str
    password: SecretStr


class UserRegister(BaseModel):
    username: str
    email: EmailStr
    password: SecretStr

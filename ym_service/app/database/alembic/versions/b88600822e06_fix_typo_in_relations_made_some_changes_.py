"""fix typo in relations, made some changes in User model

Revision ID: b88600822e06
Revises: ecd7b9c065c8
Create Date: 2025-05-29 12:29:23.012720

"""
from typing import Sequence, Union

from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision: str = 'b88600822e06'
down_revision: Union[str, None] = 'ecd7b9c065c8'
branch_labels: Union[str, Sequence[str], None] = None
depends_on: Union[str, Sequence[str], None] = None


def upgrade() -> None:
    # ### commands auto generated by Alembic - please adjust! ###
    pass
    # ### end Alembic commands ###


def downgrade() -> None:
    # ### commands auto generated by Alembic - please adjust! ###
    pass
    # ### end Alembic commands ###

"""change artist-track relation to 1-N

Revision ID: 5f1052dea0ea
Revises: 2c567dacbd26
Create Date: 2025-06-09 15:13:31.285664

"""
from typing import Sequence, Union

from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision: str = '5f1052dea0ea'
down_revision: Union[str, None] = '2c567dacbd26'
branch_labels: Union[str, Sequence[str], None] = None
depends_on: Union[str, Sequence[str], None] = None


def upgrade() -> None:
    # ### commands auto generated by Alembic - please adjust! ###
    op.drop_table('artist_track')
    op.add_column('track', sa.Column('artist_id', sa.Integer(), nullable=False))
    op.create_foreign_key(None, 'track', 'artist', ['artist_id'], ['id'])
    # ### end Alembic commands ###


def downgrade() -> None:
    # ### commands auto generated by Alembic - please adjust! ###
    op.drop_constraint(None, 'track', type_='foreignkey')
    op.drop_column('track', 'artist_id')
    op.create_table('artist_track',
    sa.Column('artist_id', sa.INTEGER(), autoincrement=False, nullable=False),
    sa.Column('track_id', sa.INTEGER(), autoincrement=False, nullable=False),
    sa.ForeignKeyConstraint(['artist_id'], ['artist.id'], name=op.f('artist_track_artist_id_fkey')),
    sa.ForeignKeyConstraint(['track_id'], ['track.id'], name=op.f('artist_track_track_id_fkey')),
    sa.PrimaryKeyConstraint('artist_id', 'track_id', name=op.f('artist_track_pkey'))
    )
    # ### end Alembic commands ###

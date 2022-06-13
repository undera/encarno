from setuptools import setup

setup(
    name="encarno",
    version="0.0",  # TODO: automate it

    author="Andrei Pokhilko",
    author_email="andrei.pokhilko@gmail.com",
    license="MIT",
    description="Python module for Taurus to use Encarno load generator",
    long_description="Python module for Taurus to use Encarno load generator",
    url='https://github.com/undera/encarno',
    keywords=[],

    packages=["encarno"],
    install_requires=[
        'bzt',
    ],
    package_data={'': ['*.json', '*.yml']},
    include_package_data=True,
)

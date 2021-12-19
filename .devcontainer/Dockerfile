FROM mcr.microsoft.com/vscode/devcontainers/base:ubuntu-21.04

ARG GOLANG_VERSION="1.17"
ARG TERRAFORM_VERSION="1.1.0"

ENV DEBIAN_FRONTEND="noninteractive"

RUN apt-get update \
    && apt-get -y install --no-install-recommends \
        build-essential make\
        fish

USER vscode

ENV HOME="/home/vscode"
ENV GOENV_ROOT="$HOME/.goenv"
ENV LANG="en_GB.UTF-8"
ENV PATH="$HOME/.tfenv/bin:$GOENV_ROOT/bin:$HOME/.local/bin:$HOME/go/$GOLANG_VERSION/bin:$PATH"

WORKDIR $HOME

RUN git clone https://github.com/syndbg/goenv.git "$HOME/.goenv" \
    && eval "$(goenv init -)" \
    && goenv install $GOLANG_VERSION \
    && goenv global $GOLANG_VERSION

RUN git clone https://github.com/tfutils/tfenv.git "$HOME/.tfenv" \
    && tfenv install $TERRAFORM_VERSION

RUN eval "$(goenv init -)" \
    && go install "github.com/aquasecurity/tfsec/cmd/tfsec@latest"

RUN eval "$(goenv init -)" \
    && git clone https://github.com/infracost/infracost.git /tmp/infracost \
    && cd /tmp/infracost \
    && make build \
    && make install \
    && cd - \
    && rm -rf /tmp/infracost

RUN git clone https://github.com/magefile/mage /tmp/mage \
    && cd /tmp/mage \
    && eval "$(goenv init -)" \
    && go run bootstrap.go \
    && cd - \
    && rm -rf /tmp/mage

RUN eval "$(goenv init -)" \
    && go install "golang.org/x/tools/gopls@latest"

RUN echo -e 'eval "$(goenv init -)"' >>$HOME/.profile

RUN mkdir -p $HOME/.config/fish \
    && touch $HOME/.config/fish/config.fish \
    && echo "status --is-interactive; and source (goenv init -|psub)" >$HOME/.config/fish/config.fish

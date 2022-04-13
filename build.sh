#!/bin/bash

# root path of cb-larva
echo "1. Set CBLARVA_ROOT path"
sleep 2
SCRIPT_DIR=`dirname ${BASH_SOURCE[0]-$0}`
export CBLARVA_ROOT=`cd $SCRIPT_DIR && pwd`
echo ${CBLARVA_ROOT}
echo ""

# create directory
echo "2. Create directories for binaries and assets"
sleep 2
mkdir ${CBLARVA_ROOT}/bin
mkdir ${CBLARVA_ROOT}/bin/config
mkdir ${CBLARVA_ROOT}/bin/web
mkdir ${CBLARVA_ROOT}/bin/docs

echo "tree -L 2 -N ${CBLARVA_ROOT}/bin"
sleep 2
tree -L 2 -N ${CBLARVA_ROOT}/bin
echo ""

# build all cb-network system components
echo "3. Build binaries"
sleep 2
cd ${CBLARVA_ROOT}/poc-cb-net
make
echo ""

# copy binaries and assets to 'bin'
echo "4. Copy binaries and assets to 'bin'"
sleep 2
cd ${CBLARVA_ROOT}/bin

# copy cb-network controller binary
cp ${CBLARVA_ROOT}/poc-cb-net/cmd/controller/controller ./

# copy service binary and asset
cp ${CBLARVA_ROOT}/poc-cb-net/cmd/service/service ./
cp ${CBLARVA_ROOT}/poc-cb-net/docs/cloud_barista_network.swagger.json ./docs/

# copy cb-network admin-web binary and assets
cp ${CBLARVA_ROOT}/poc-cb-net/cmd/admin-web/admin-web ./
cp -r ${CBLARVA_ROOT}/poc-cb-net/web/* ./web/

# copy config files
cp ${CBLARVA_ROOT}/poc-cb-net/config/template-config.yaml ./config/config.yaml
cp ${CBLARVA_ROOT}/poc-cb-net/config/template-log_conf.yaml ./config/log_conf.yaml

echo "tree -L 2 -N ${CBLARVA_ROOT}/bin"
sleep 2
tree -L 2
echo ""

echo "Done to build"
sleep 2
echo ""
echo "[Note] Please, edit 'config.yaml' and 'log_conf.yaml' before running binaries"
echo "[Note] Please, edit 'config.yaml' and 'log_conf.yaml' before running binaries"
echo "[Note] Please, edit 'config.yaml' and 'log_conf.yaml' before running binaries"

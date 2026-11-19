#!/usr/bin/env bash
set -e

DIR=$(dirname "$0")
NM="${DIR}/node_modules"
OUT="${DIR}/assets"

# Install scripts are skipped: summernote (pulled in by admin-lte, never shipped) runs husky on install.
npm ci --omit=dev --ignore-scripts --no-audit --no-fund --prefix "${DIR}"

rm -rf "${OUT}/dist" "${OUT}/plugins" "${OUT}/custom"

mkdir -p "${OUT}/dist/js" "${OUT}/dist/css"
cp "${NM}/admin-lte/dist/js/adminlte.min.js" "${OUT}/dist/js/"
cp "${NM}/admin-lte/dist/css/adminlte.min.css" "${OUT}/dist/css/"

cp -r "${DIR}/custom" "${OUT}/"

P="${OUT}/plugins"
mkdir -p "${P}/jquery" "${P}/bootstrap/js" "${P}/fontawesome-free" "${P}/icheck-bootstrap" \
  "${P}/jquery-validation" "${P}/select2/css" "${P}/select2/js" "${P}/toastr" "${P}/jquery-tags-input/dist"

cp "${NM}/jquery/dist/jquery.min.js" "${P}/jquery/"
cp "${NM}/bootstrap/dist/js/bootstrap.bundle.min.js" "${P}/bootstrap/js/"
cp -r "${NM}/@fortawesome/fontawesome-free/css" "${NM}/@fortawesome/fontawesome-free/webfonts" "${P}/fontawesome-free/"
cp "${NM}/icheck-bootstrap/icheck-bootstrap.min.css" "${P}/icheck-bootstrap/"
cp "${NM}/jquery-validation/dist/jquery.validate.min.js" "${P}/jquery-validation/"
cp "${NM}/select2/dist/css/select2.min.css" "${P}/select2/css/"
cp "${NM}/select2/dist/js/select2.full.min.js" "${P}/select2/js/"
cp "${NM}/toastr/build/toastr.min.css" "${NM}/toastr/build/toastr.min.js" "${P}/toastr/"
cp "${NM}/jquery-tags-input/dist/jquery.tagsinput.min.css" "${NM}/jquery-tags-input/dist/jquery.tagsinput.min.js" "${P}/jquery-tags-input/dist/"

<?php
/**
 * iF.SVNAdmin
 * Copyright (c) 2010 by Manuel Freiholz
 * http://www.insanefactory.com/
 *
 * This program is free software; you can redistribute it and/or
 * modify it under the terms of the GNU General Public License
 * as published by the Free Software Foundation; version 2
 * of the License.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.
 */
namespace SvnAdmin\Contracts;

interface IRepositoryEditProvider extends IEditProvider
{
	/**
	 * Creates a new repository.
	 *
	 * @param \SvnAdmin\Entities\Repository   $oRepository
	 *
	 * @return bool
	 */
	public function create(\SvnAdmin\Entities\Repository $oRepository);

	/**
	 * Deletes an existing repository.
	 *
	 * @param \SvnAdmin\Entities\Repository $oRepository
	 *
	 * @return bool
	 */
	public function delete(\SvnAdmin\Entities\Repository $oRepository);

	/**
	 * Creates a new folder in the repository (including parents of folder).
	 *
	 * @param \SvnAdmin\Entities\Repository $oRepository
	 * @param array $path Paths to the folders which should be created.
	 *
	 * @return bool
	 */
	public function mkdir(\SvnAdmin\Entities\Repository $oRepository, array $paths);
}
?>